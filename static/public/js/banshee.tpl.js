angular.module("banshee.tpl", []).run(["$templateCache", function($templateCache) {$templateCache.put("modules/layout/footer.html","<p>Built and maintained by <a href=https://github.com/hit9>hit9</a>. Code licensed under MIT (c) Eleme, Inc and hosted on <a href=https://github.com/eleme/bell.js>Github</a>.</p>");
$templateCache.put("modules/layout/header.html","<nav role=navigation class=\"navbar topnavbar\"><div class=navbar-header><a href=\"#/\" class=navbar-brand><div class=brand-logo></div></a></div><div class=nav-wrapper><ul class=\"nav navbar-nav navbar-right\"><li><a href=javascript:;>Version2.0.3</a></li><li><a href=#>Administration</a></li></ul></div></nav>");
$templateCache.put("modules/layout/index.html","<div class=wrapper><header ui-view=header class=topnavbar-wrapper></header><aside ui-view=navigation class=aside></aside><section><div ui-view class=content-wrapper></div></section><footer ui-view=footer></footer></div>");
$templateCache.put("modules/layout/navigation.html","<div class=aside-inner><nav class=sidebar><ul class=nav><li class=active><a title=Navigation class=ng-scope><em class=icon-map></em> <span>Navigation</span></a><ul class=\"nav sidebar-subnav collapse in\" style=\"height: auto\"><li class=sidebar-subnav-header>Navigation</li><li><a ui-sref=\"/\" title=Project><span>Project</span></a></li><li><a ui-sref=\"/\" title=Receiver><span>Receiver</span></a></li><li><a ui-sref=\"/\" title=Configuration><span>Configuration</span></a></li></ul></li></ul></nav></div>");
$templateCache.put("modules/admin/project/list.html","<div id=admin-project-list><div class=col-lg-6><div class=\"panel panel-default\"><div class=panel-heading>Project <button class=\"btn btn-labeled btn-success pull-right\" ng-click=openModal()><span class=btn-label><i class=icon-plus></i></span> Create</button></div><div class=panel-body><div class=table-responsive><table class=table><thead><tr><th class=th50>ID</th><th>Project Name</th><th class=th100>Options</th></tr></thead><tbody><tr ng-repeat=\"project in projects\"><td ng-bind=project.id></td><td ng-bind=project.name></td><td><a ui-sref=banshee.admin class=\"btn btn-sm btn-info\"><i class=icon-bubble></i></a> <a ui-sref=banshee.admin class=\"btn btn-sm btn-green\"><i class=icon-pencil></i></a></td></tr></tbody></table></div></div></div></div></div>");
$templateCache.put("modules/admin/project/projectModal.html","<form name=form class=form-horizontal ng-submit=create() onvalidate><div class=modal-header><h4 class=\"modal-title inline-block\">Create Project</h4></div><div class=\"modal-body row\"><div class=\"form-group col-md-10\"><label for=input-id-1 class=\"col-md-3 control-label\">Project Name</label><div class=col-md-9><input name=projectName class=form-control ng-model=project.name required> <small ng-if=\"form.projectName.$dirty && form.projectName.$error.required\" class=\"help-block color-red\">This field is required.</small></div></div></div><div class=\"modal-footer text-right\"><button class=\"btn btn-default\" type=button ng-click=$dismiss()>cancel</button> <button class=\"btn btn-primary\" type=submit ng-disabled=form.$invalid>submit</button></div></form>");}]);