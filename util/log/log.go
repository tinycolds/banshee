// Copyright 2015 Eleme Inc. All rights reserved.

// Package log implements leveled logging.
package log

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"time"
)

// Level
const (
	DEBUG int = iota
	INFO
	WARN
	ERROR
)

// Level name
var levelNames = [4]string{"DEBUG", "INFO", "WARN", "ERROR"}

// Logging runtime
var (
	name    string
	level             = INFO
	w       io.Writer = os.Stderr
	colored           = true
	enabled           = true
)

// colors to ansi code map
var colors = map[string]int{
	"black":   0,
	"red":     1,
	"green":   2,
	"yellow":  3,
	"blue":    4,
	"magenta": 5,
	"cyan":    6,
	"white":   7,
}

// levelColors
var levelColors = map[int]string{
	DEBUG: "cyan",
	INFO:  "white",
	WARN:  "yellow",
	ERROR: "red",
}

// SetColored sets the color enability.
func SetColored(b bool) {
	colored = b
}

// SetName sets the logging name.
func SetName(s string) {
	name = s
}

// SetLevel sets the logging level.
func SetLevel(l int) {
	level = l % len(levelNames)
}

// SetWriter sets the writer.
func SetWriter(writer io.Writer) {
	w = writer
}

// Disable the logging.
func Disable() {
	enabled = false
}

// Enable the logging.
func Enable() {
	enabled = true
}

// Debug logs message with level DEBUG.
func Debug(format string, a ...interface{}) {
	log(DEBUG, format, a...)
}

// Info logs message with level INFO.
func Info(format string, a ...interface{}) {
	log(INFO, format, a...)
}

// Warn logs message with level WARN.
func Warn(format string, a ...interface{}) {
	log(WARN, format, a...)
}

// Error logs message with level ERROR.
func Error(format string, a ...interface{}) {
	log(ERROR, format, a...)
}

// Fatal logs message with level FATAL.
func Fatal(format string, a ...interface{}) {
	log(ERROR, format, a...)
	os.Exit(1)
}

// Colored returns text in color.
func Colored(color string, text string) string {
	return fmt.Sprintf("\033[3%dm%s\033[0m", colors[color], text)
}

// log dose logging.
func log(l int, format string, a ...interface{}) {
	if enabled && l >= level {
		// Caller pkg.
		_, fileName, _, _ := runtime.Caller(2)
		pkgName := path.Base(path.Dir(fileName))
		// Datetime and pid.
		now := time.Now().String()[:23]
		pid := os.Getpid()
		// Message
		msg := fmt.Sprintf(format, a...)
		var (
			slevel = fmt.Sprintf("%-5s", levelNames[l])
			sname  = fmt.Sprintf("%s.%-8s", name, pkgName)
		)
		if colored {
			sname = Colored("white", sname)
			slevel = Colored(levelColors[l], slevel)
		}
		s := fmt.Sprintf("%s %s %s[%d]: %s", now, slevel, sname, pid, msg)
		fmt.Fprintln(w, s)
	}
}
