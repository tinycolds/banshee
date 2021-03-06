// Copyright 2015 Eleme Inc. All rights reserved.

package detector

import (
	"bufio"
	"fmt"
	"net"
	"path/filepath"

	"github.com/eleme/banshee/config"
	"github.com/eleme/banshee/filter"
	"github.com/eleme/banshee/health"
	"github.com/eleme/banshee/models"
	"github.com/eleme/banshee/storage"
	"github.com/eleme/banshee/storage/indexdb"
	"github.com/eleme/banshee/util"
	"github.com/eleme/banshee/util/log"
	"github.com/eleme/banshee/util/mathutil"
)

// Timeout in milliseconds.
const timeout = 300

// Detector is to detect anomalies.
type Detector struct {
	cfg  *config.Config
	db   *storage.DB
	flt  *filter.Filter
	outs []chan *models.Metric
}

// New creates a detector.
func New(cfg *config.Config, db *storage.DB, flt *filter.Filter) *Detector {
	return &Detector{cfg, db, flt, make([]chan *models.Metric, 0)}
}

// Out adds a channel to receive detection results.
func (d *Detector) Out(ch chan *models.Metric) {
	d.outs = append(d.outs, ch)
}

// Output detected metrics to channels in outs, will skip if the target channel
// is full.
func (d *Detector) output(m *models.Metric) {
	for _, ch := range d.outs {
		select {
		case ch <- m:
		default:
			log.Error("output channel is full, skipping..")
			continue
		}
	}
}

// Start the tcp server.
func (d *Detector) Start() {
	// Listen
	addr := fmt.Sprintf("0.0.0.0:%d", d.cfg.Detector.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("listen: %v", err)
	}
	log.Info("detector is listening on %s..", addr)
	// Accept
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Error("cannot accept conn: %v, skipping..", err)
			continue
		}
		go d.handle(conn)
	}
}

// Handle a new connection, it will:
//
//	1. Read input from the connection line by line.
//	2. Parse the lines into metrics.
//	3. Validate the metrics.
//
func (d *Detector) handle(conn net.Conn) {
	// New conn established.
	addr := conn.RemoteAddr()
	health.IncrNumClients(1)
	log.Info("conn %s established", addr)
	// Read
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		// Read line by line.
		if err := scanner.Err(); err != nil {
			// Close conn on read error.
			log.Error("read error: %v, closing conn..", err)
			break
		}
		line := scanner.Text()
		// Parse metric.
		m, err := parseMetric(line)
		if err != nil {
			// Skip invalid input.
			log.Error("parse error: %v, skipping..", err)
			continue
		}
		// Validate metric.
		if err := models.ValidateMetricName(m.Name); err != nil {
			log.Error("invalid metric: %v, skipping..", err)
			continue
		}
		if err := models.ValidateMetricStamp(m.Stamp); err != nil {
			log.Error("invalid metric: %v, skipping..", err)
			continue
		}
		// Process
		d.process(m)
	}
	// Close conn.
	conn.Close()
	log.Info("conn %s disconnected", addr)
	health.DecrNumClients(1)
}

// Process the input metric.
//
//	1. Match metric with rules.
//	2. Detect the metric with matched rules.
//
func (d *Detector) process(m *models.Metric) {
	health.IncrNumMetricIncomed(1)
	timer := util.NewTimer()
	// Match
	ok, rules := d.match(m)
	if !ok {
		// Not matched.
		return
	}
	// Detect
	err := d.detect(m, rules)
	if err != nil {
		log.Error("detect: %v, skipping..", err)
		return
	}
	health.IncrNumMetricDetected(1)
	// Output
	if len(m.TestedRules) > 0 {
		// Test ok.
		d.output(m)
	}
	// Time end.
	elapsed := timer.Elapsed()
	if elapsed > timeout {
		log.Warn("detection is slow: %.2fms", elapsed)
	}
	health.AddDetectionCost(elapsed)
}

// Match a metric with rules, and return matched rules.
//
//	If no rules matched, return false.
//	If any black patterns matched, return false.
//	Else, return true and matched rules.
//
func (d *Detector) match(m *models.Metric) (bool, []*models.Rule) {
	// Check rules.
	timer := util.NewTimer()
	rules := d.flt.MatchedRules(m)
	elapsed := timer.Elapsed()
	health.AddFilterCost(elapsed)
	if len(rules) == 0 {
		// Hit no rules.
		return false, rules
	}
	// Check blacklist.
	for _, p := range d.cfg.Detector.BlackList {
		ok, err := filepath.Match(p, m.Name)
		if err != nil {
			// Invalid black pattern.
			log.Error("invalid black pattern: %s, %v", p, err)
			continue
		}
		if ok {
			// Hit black pattern.
			log.Debug("%s hit black pattern %s", m.Name, p)
			return false, rules
		}
	}
	// Ok
	return true, rules
}

// Detect input metric with its matched rules via 3-sigma.
//
//	1. Get history values for this metric.
//	2. Get current index for this metric.
//	3. Calculate score via 3-sigma.
//	4. Get score trending via ewma.
//	5. Save the metric and index to db.
//	6. Test with its matched rules and output it.
//
func (d *Detector) detect(m *models.Metric, rules []*models.Rule) error {
	// Get index.
	idx, err := d.db.Index.Get(m.Name)
	if err != nil {
		if err == indexdb.ErrNotFound {
			idx = nil
		} else {
			return err // unexcepted
		}
	}
	// Fill zero?
	fz := idx != nil && d.shouldFz(m)
	// History values.
	vals, err := d.values(m, fz)
	if err != nil {
		return err // unexcepted
	}
	// Apply 3-sigma.
	d.div3Sigma(m, vals)
	// New index.
	idx = d.nextIdx(idx, m)
	// Test with rules.
	d.test(m, idx, rules)
	// Save
	return d.save(m, idx)
}

// Test metric and index with rules.
// The following function will fill the m.TestedRules.
func (d *Detector) test(m *models.Metric, idx *models.Index, rules []*models.Rule) {
	for _, rule := range rules {
		if rule.Test(m, idx, d.cfg) {
			// Add tested ok rules.
			m.TestedRules = append(m.TestedRules, rule)
		}
	}
}

// Save metric and index into db.
func (d *Detector) save(m *models.Metric, idx *models.Index) error {
	// Save index.
	if err := d.db.Index.Put(idx); err != nil {
		return err
	}
	// Save metric.
	if err := d.db.Metric.Put(m); err != nil {
		return err
	}
	return nil
}

// Test whether a metric need to fill blank with zeros to its history
// values.
func (d *Detector) shouldFz(m *models.Metric) bool {
	for _, p := range d.cfg.Detector.FillBlankZeros {
		ok, err := filepath.Match(p, m.Name)
		if err != nil {
			// Invalid pattern.
			log.Error("invalid fillBlankZeros pattern: %s, %v", p, err)
			continue
		}
		if ok {
			// Ok.
			return true
		}
	}
	// No need.
	return false
}

// Fill blank with zeros into history values, mainly for dispersed
// metrics such as counters. The start and stop is for periodicity
// reasons.
func (d *Detector) fill0(ms []*models.Metric, start, stop uint32) []float64 {
	i := 0 // record real-metric.
	step := d.cfg.Interval
	var vals []float64
	for start < stop {
		if i < len(ms) {
			m := ms[i]
			// start is smaller than current stamp.
			for ; start < m.Stamp; start += step {
				vals = append(vals, 0)
			}
			// Push real-metric.
			vals = append(vals, m.Value)
			i++
		} else {
			// No more real-metric.
			vals = append(vals, 0)
		}
		start += step
	}
	return vals
}

// Result struct help to receive multiple return values.
type metricGetResult struct {
	err   error
	ms    []*models.Metric
	start uint32
	stop  uint32
}

// Get history values for the input metric, will only fetch the history
// values with the same phase around this timestamp, within an filter
// offset.
func (d *Detector) values(m *models.Metric, fz bool) ([]float64, error) {
	offset := uint32(d.cfg.Detector.FilterOffset * float64(d.cfg.Period))
	expiration := d.cfg.Expiration
	period := d.cfg.Period
	// Get values with the same phase.
	n := 0
	ch := make(chan metricGetResult)
	for stamp := m.Stamp; stamp+expiration > m.Stamp; stamp -= period {
		start := stamp - offset
		stop := stamp + offset
		n++
		go func() {
			ms, err := d.db.Metric.Get(m.Name, start, stop)
			ch <- metricGetResult{err, ms, start, stop}
		}()
	}
	// Concat chunks.
	var vals []float64
	var err error
	for i := 0; i < n; i++ {
		r := <-ch
		if r.err != nil {
			// Record error but DONOT return directly.
			// Must receive n times from ch, otherwise the goroutine will
			// be hanged and the ch won't be gc, yet memory leaks.
			err = r.err
			continue
		}
		if err != nil {
			continue
		}
		// Append to values.
		if !fz {
			for j := 0; j < len(r.ms); j++ {
				vals = append(vals, r.ms[j].Value)
			}
		} else {
			// Fill blank with zeros.
			vals = append(vals, d.fill0(r.ms, r.start, r.stop)...)
		}
	}
	if err != nil {
		// Unexcepted error
		return vals, err
	}
	// Append m
	vals = append(vals, m.Value)
	return vals, nil
}

// Calculate metric score with 3-sigma rule.
//
// What's the 3-sigma rule?
//
//	states that nearly all values (99.7%) lie within the 3 standard deviations
//	of the mean in a normal distribution.
//
// Also like z-score, defined as
//
//	(val - mean) / stddev
//
// And we name the below as metric score, yet 1/3 of z-score
//
//	(val - mean) / (3 * stddev)
//
// The score has
//
//	score > 0   => values is trending up
//	score < 0   => values is trending down
//	score > 1   => values is anomalously trending up
//	score < -1  => values is anomalously trending down
//
// The following function will set the metric score and also the average.
//
func (d *Detector) div3Sigma(m *models.Metric, vals []float64) {
	if len(vals) == 0 {
		// Values empty.
		m.Score = 0
		m.Average = m.Value
		return
	}
	// Values average and standard deviation.
	avg := mathutil.Average(vals)
	std := mathutil.StdDev(vals, avg)
	// Set metric average
	m.Average = avg
	// Set metric score
	if len(vals) <= int(d.cfg.Detector.LeastCount) {
		// Values not enough.
		m.Score = 0
		return
	}
	last := vals[len(vals)-1]
	if std == 0 {
		switch {
		case last == avg:
			m.Score = 0
		case last > avg:
			m.Score = 1
		case last < avg:
			m.Score = -1
		}
		return
	}
	// 3-sigma
	m.Score = (last - avg) / (3 * std)
}

// Calculate next score for index via ewma, called the weighted exponential
// moving average.
//
//	t[0] = x[1], f: 0~1
//	t[n] = t[n-1] * (1 - f) + f * x[n]
//
// The index score will follow the metric's score, and additionally the index
// average is the latest metric average.
//
// Index score is the trending description of metric score.
//
func (d *Detector) nextIdx(idx *models.Index, m *models.Metric) *models.Index {
	n := &models.Index{Name: m.Name, Stamp: m.Stamp}
	if idx == nil {
		// As first
		n.Score = m.Score
		n.Average = m.Value
		return n
	}
	// Move next
	f := d.cfg.Detector.TrendingFactor
	n.Score = idx.Score*(1-f) + f*m.Score
	n.Average = m.Average
	return n
}
