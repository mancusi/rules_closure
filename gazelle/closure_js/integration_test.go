package closure_js_test

import (
	"flag"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/testtools"
)

// TestJS_RulePerFile tests the JS rule generation.
func TestJS_RulePerFile(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path: "WORKSPACE",
		}, {
			Path: "BUILD.bazel",
			Content: `
# gazelle:js_grep_extern React //js/externs:react
# gazelle:js_rule_per_file true
`,
		}, {
			Path:    "corp.js",
			Content: `goog.provide('corp')`,
		}, {
			Path: "i18n.js",
			Content: `goog.provide("corp.i18n");
goog.provide('corp.msg');

goog.require('corp');
goog.require('goog.string');
goog.require('goog.i18n.MessageFormat');
`,
		},
		{
			Path: "ui/widget.jsx",
			Content: `
goog.module('corp.ui.widget');

goog.require('corp.i18n');
goog.require('goog.ui.Component');
`,
		},
		{
			Path: "ui/widget_test.jsx",
			Content: `
goog.module('corp.ui.widget.test');

goog.require('corp');
const msg = goog.require('corp.msg');
goog.require('goog.ui.Component');
var testtools = goog.require('corp.ui.widget.testtools');
		`,
		},
		{
			Path:    "ui/widget_test.html",
			Content: `<!DOCTYPE html><html></html>`,
		},
		{
			Path: "app/app.js",
			Content: `
goog.module('corp.app.App');

const msg = goog.require('corp.msg');
goog.require('corp');
const widget = goog.require('corp.ui.widget');

React.renderElement(widget);
		`,
		},
		{
			Path: "app/app_test.js",
			Content: `
goog.module('corp.app.AppTest');

const app = goog.require('corp.app.App');
		`,
		},
		{
			Path: "existing/file1.js",
			Content: `goog.module('corp.existing.file1');
goog.require('corp.msg');
goog.require('goog.string');
`,
		},
		{
			Path: "existing/file2.js",
			Content: `goog.module('corp.existing.file1');
goog.require('corp.ui.widget');
goog.require('goog.dom.query');
const file1 = goog.require('corp.existing.file1');
`,
		},
		{
			Path: "existing/file3.js",
			Content: `goog.module('corp.existing.file3');
goog.require('goog.array');
const file1 = goog.require('corp.existing.file1');
`,
		},
		{
			Path:    "existing/file4.js",
			Content: `goog.module('corp.existing.file4');`,
		},
		{
			Path: "existing/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "existing",
    srcs = ["file1.js", "file2.js"],
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "existingfile3",
    srcs = ["file3.js"],
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "deleted",
    srcs = ["deleted.js"],
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "deleted_multifile",
    srcs = ["deleted1.js", "deleted2.js"],
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "deleted_some_multifile",
    srcs = ["file4.js", "deleted3.js"],
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "globbed",
    srcs = glob(["*.js"], exclude=["file1.js", "file2.js", "file3.js"]),
    visibility = ["//visibility:public"],
)
`,
		},
		{
			Path:    "map_kind/file1.js",
			Content: `goog.module('corp.map_kind.file1');`,
		},
		{
			Path:    "map_kind/file2.js",
			Content: `goog.module('corp.map_kind.file2');`,
		},
		{
			Path: "map_kind/BUILD.bazel",
			Content: `
load("//tools/js:defs.bzl", "closure_js_extern_library")

# gazelle:map_kind closure_js_library closure_js_extern_library //tools/js:defs.bzl

closure_js_extern_library(
    name = "existing",
    srcs = ["file1.js"],
    visibility = ["//visibility:public"],
)

closure_js_extern_library(
    name = "deleted",
    srcs = ["deleted.js"],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "js/bugs/dupe_dep_not_caused_by_grep_extern_and_requires.js",
			Content: `
goog.require('React');

React.renderElement();
`,
		},
		{
			Path: "js/externs/react.js",
			Content: `/** @externs */
goog.provide('React');
var React = {};`,
		},

		{
			Path: "js/bugs/dupe_externs/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

# gazelle:js_grep_extern $ //js/externs:jquery

closure_js_library(
    name = "combined",
    srcs = [
        "a.js",
        "b.js",
    ],
)
`,
		},
		{
			Path:    "js/bugs/dupe_externs/a.js",
			Content: `($)`,
		},
		{
			Path:    "js/bugs/dupe_externs/b.js",
			Content: `($)`,
		},

		// ES6 tests
		{
			Path:    "es6modules/utils/display-utils.jsx",
			Content: ``,
		},
		{
			Path: "es6modules/app/fields/widget.jsx",
			Content: `
import { IndeterminateValue } from '../../utils/display-utils';
import widget from 'goog:corp.ui.widget';
import { capitalize } from 'goog:goog.string';
const { moveItem } = goog.require('goog.array');
goog.require('corp.i18n');
`,
		},
		{
			Path: "es6modules/app/fields/widget_test.jsx",
			Content: `
import { Widget } from '/es6modules/app/fields/widget';
import testSuite from 'goog:goog.testing.testSuite';
 `,
		},
	}
	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	if err := runGazelle(t, dir, []string{}); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path: "BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

# gazelle:js_grep_extern React //js/externs:react
# gazelle:js_rule_per_file true

closure_js_library(
    name = "corp",
    srcs = ["corp.js"],
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "i18n",
    srcs = ["i18n.js"],
    visibility = ["//visibility:public"],
    deps = [
        ":corp",
        "@io_bazel_rules_closure//closure/library/i18n:messageformat",
        "@io_bazel_rules_closure//closure/library/string",
    ],
)
`,
		},
		{
			Path: "ui/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_jsx_library", "closure_jsx_test")

closure_jsx_library(
    name = "widget",
    srcs = ["widget.jsx"],
    visibility = ["//visibility:public"],
    deps = [
        "//:i18n",
        "@io_bazel_rules_closure//closure/library/ui:component",
    ],
)

closure_jsx_test(
    name = "widget_test",
    srcs = ["widget_test.jsx"],
    compilation_level = "ADVANCED",
    entry_points = ["corp.ui.widget.test"],
    html = "widget_test.html",
    visibility = ["//visibility:public"],
    deps = [
        "//:corp",
        "//:i18n",
        "@io_bazel_rules_closure//closure/library/ui:component",
    ],
)
`,
		},
		{
			Path: "app/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library", "closure_js_test")

closure_js_library(
    name = "app",
    srcs = ["app.js"],
    visibility = ["//visibility:public"],
    deps = [
        "//:corp",
        "//:i18n",
        "//js/externs:react",
        "//ui:widget",
    ],
)

closure_js_test(
    name = "app_test",
    srcs = ["app_test.js"],
    compilation_level = "ADVANCED",
    entry_points = ["corp.app.AppTest"],
    visibility = ["//visibility:public"],
    deps = [":app"],
)
`,
		},
		{
			Path: "existing/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "existing",
    srcs = [
        "file1.js",
        "file2.js",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//:i18n",
        "//ui:widget",
        "@io_bazel_rules_closure//closure/library/string",
        "@io_bazel_rules_closure//third_party/closure/library/dojo/dom:query",
    ],
)

closure_js_library(
    name = "existingfile3",
    srcs = ["file3.js"],
    visibility = ["//visibility:public"],
    deps = [
        ":existing",
        "@io_bazel_rules_closure//closure/library/array",
    ],
)

closure_js_library(
    name = "deleted_some_multifile",
    srcs = ["file4.js"],
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "globbed",
    srcs = glob(
        ["*.js"],
        exclude = [
            "file1.js",
            "file2.js",
            "file3.js",
        ],
    ),
    visibility = ["//visibility:public"],
)
`,
		},
		{
			Path: "map_kind/BUILD.bazel",
			Content: `
load("//tools/js:defs.bzl", "closure_js_extern_library")

# gazelle:map_kind closure_js_library closure_js_extern_library //tools/js:defs.bzl

closure_js_extern_library(
    name = "existing",
    srcs = ["file1.js"],
    visibility = ["//visibility:public"],
)

closure_js_extern_library(
    name = "file2",
    srcs = ["file2.js"],
    visibility = ["//visibility:public"],
)
`,
		},
		{
			Path: "js/externs/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "react",
    srcs = ["react.js"],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "js/bugs/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "dupe_dep_not_caused_by_grep_extern_and_requires",
    srcs = ["dupe_dep_not_caused_by_grep_extern_and_requires.js"],
    visibility = ["//visibility:public"],
    deps = ["//js/externs:react"],
)
`,
		},

		{
			Path: "js/bugs/dupe_externs/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

# gazelle:js_grep_extern $ //js/externs:jquery

closure_js_library(
    name = "combined",
    srcs = [
        "a.js",
        "b.js",
    ],
    visibility = ["//visibility:public"],
    deps = ["//js/externs:jquery"],
)
`,
		},

		{
			Path: "es6modules/utils/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_jsx_library")

closure_jsx_library(
    name = "display-utils",
    srcs = ["display-utils.jsx"],
    visibility = ["//visibility:public"],
)
`,
		},
		{
			Path: "es6modules/app/fields/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_jsx_library", "closure_jsx_test")

closure_jsx_library(
    name = "widget",
    srcs = ["widget.jsx"],
    visibility = ["//visibility:public"],
    deps = [
        "//:i18n",
        "//es6modules/utils:display-utils",
        "//ui:widget",
        "@io_bazel_rules_closure//closure/library/array",
        "@io_bazel_rules_closure//closure/library/string",
    ],
)

closure_jsx_test(
    name = "widget_test",
    srcs = ["widget_test.jsx"],
    compilation_level = "ADVANCED",
    entry_points = ["/es6modules/app/fields/widget_test"],
    visibility = ["//visibility:public"],
    deps = [
        ":widget",
        "@io_bazel_rules_closure//closure/library/testing:testsuite",
    ],
)
`,
		},
	})
}

// TestJSVisibility tests the JS rule generation's visibility attribute.
func TestJSVisibility(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path: "WORKSPACE",
		},

		// Test that a package with default visibility results in no override on
		// the individual rule.
		{
			Path: "BUILD.bazel",
			Content: `
package(default_visibility = ["//a:__subpackages__"])

# gazelle:js_rule_per_file
`,
		}, {
			Path:    "corp.js",
			Content: `goog.provide('corp')`,
		},

		// Test that a package under internal has visibility set correctly.
		{
			Path:    "a/b/internal/internal.js",
			Content: `goog.provide('a.b.internal')`,
		},
		{
			Path:    "a/b/internal/c/d/internal.js",
			Content: `goog.provide('a.b.c.d.internal')`,
		},
		{
			Path:    "a/b/internal/c/internal/d/internal.js",
			Content: `goog.provide('a.b.c.internal.d.internal')`,
		},
	}

	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	if err := runGazelle(t, dir, []string{}); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path: "BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

package(default_visibility = ["//a:__subpackages__"])

# gazelle:js_rule_per_file

closure_js_library(
    name = "corp",
    srcs = ["corp.js"],
)
`,
		},

		{
			Path: "a/b/internal/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "internal",
    srcs = ["internal.js"],
    visibility = ["//a/b:__subpackages__"],
)
		`,
		},

		{
			Path: "a/b/internal/c/d/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "internal",
    srcs = ["internal.js"],
    visibility = ["//a/b:__subpackages__"],
)
`,
		},

		{
			Path: "a/b/internal/c/internal/d/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "internal",
    srcs = ["internal.js"],
    visibility = ["//a/b/internal/c:__subpackages__"],
)
`,
		},
	})

}

func TestJSRulePerDirectory(t *testing.T) {
	files := []testtools.FileSpec{
		{
			Path: "WORKSPACE",
		},

		// Test that a package with no JS files gets no files.
		{Path: "no_js/BUILD.bazel"},
		{Path: "no_js/README"},

		// Test that a package with no rules has files combined into a directory.
		{
			Path:    "no_rules/norules.js",
			Content: `goog.provide('no_rules')`,
		},

		// Test that js files at the top level are called "lib" instead of the
		// name of random directory it's in.
		{Path: "toplevel.js"},

		// Test that default visibility is respected.
		{
			Path:    "default_vis/BUILD.bazel",
			Content: `package(default_visibility = ["//a:__subpackages__"])`,
		}, {
			Path: "default_vis/defaultvis.js",
		},

		// Test that a package with multiple files has them combined.
		{Path: "no_rules/multiple_files/a.js"},
		{Path: "no_rules/multiple_files/b.js"},
		{Path: "no_rules/multiple_files/c.js"},

		// Test that a rule that includes a jsx file uses closure_jsx_library.
		{Path: "no_rules/multiple_files_jsx/a.js"},
		{Path: "no_rules/multiple_files_jsx/b.jsx"},
		{Path: "no_rules/multiple_files_jsx/c.js"},

		// Test that a rule includes all deps of sources, de-duplicated.
		// Test that a intra-library dependency is handled without a dep.
		// Test that externs are combined and de-duplicated.
		{
			Path: "no_rules/combine_deps/BUILD.bazel",
			Content: `
# gazelle:js_grep_extern React //js/externs:react
# gazelle:js_grep_extern $ //js/externs:jQuery
# gazelle:js_grep_extern _ //js/externs:lodash
`,
		},
		{
			Path: "no_rules/combine_deps/a.js",
			Content: `
goog.provide('no_rules.combine_deps.a');

goog.require('goog.array');
goog.require('goog.string');

React($)
`,
		},
		{
			Path: "no_rules/combine_deps/b.jsx",
			Content: `
goog.provide('no_rules.combine_deps.b');

goog.require('goog.dom.query');
goog.require('goog.string');
goog.require('no_rules.combine_deps.a');

React(_)
`,
		},

		// Test that a package with existing rules (but no directory-level rule)
		// maintains the existing ones and adds new files to a library-wide
		// rule.
		{
			Path: "existing_rules/file1.js",
			Content: `goog.module('corp.existing.file1');
goog.require('corp.msg');
goog.require('goog.string');
`,
		},
		{
			Path: "existing_rules/file2.js",
			Content: `goog.module('corp.existing.file1');
goog.require('goog.dom.query');
const file1 = goog.require('corp.existing.file1');
`,
		},
		{
			Path: "existing_rules/file3.js",
			Content: `goog.module('corp.existing.file3');
goog.require('goog.array');
const file1 = goog.require('corp.existing.file1');
`,
		},
		{
			Path:    "existing_rules/file4.js",
			Content: `goog.module('corp.existing.file4');`,
		},
		{Path: "existing_rules/file5.js"},
		{Path: "existing_rules/file6.js"},
		{Path: "existing_rules/testdata"},
		{
			Path: "existing_rules/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "existing",
    srcs = ["file1.js", "file2.js"],
    visibility = ["//visibility:public"],
    data = ["testdata"],
    deps = [
        ":testdep", # keep
    ]
)

closure_js_library(
    name = "existingfile3",
    srcs = ["file3.js"],
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "deleted",
    srcs = ["deleted.js"],
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "deleted_multifile",
    srcs = ["deleted1.js", "deleted2.js"],
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "deleted_some_multifile",
    srcs = ["file4.js", "deleted3.js"],
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "globbed",
    srcs = glob(["*.js"], exclude=["file1.js", "file2.js", "file3.js"]),
    visibility = ["//visibility:public"],
)
`,
		},

		// Test that a package with an existing directory-level rule adds to it.
		// Test that dependencies of individual files are combined.
		{
			Path: "existing_lib/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "existing_lib",
    srcs = ["file1.js"],
    visibility = ["//visibility:public"],
)
`,
		},
		{Path: "existing_lib/file1.js", Content: "goog.require('goog.array');"},
		{Path: "existing_lib/file2.js", Content: "goog.require('goog.string');"},

		// Test resolving imports to rules in a directory-level lib.
		{Path: "resolve/BUILD.bazel"},
		{Path: "resolve/file1.js", Content: "goog.module('resolve.file1');"},
		{Path: "resolve/file2.js", Content: "goog.module('resolve.file2');"},
		{Path: "resolve/other/BUILD.bazel", Content: "# gazelle:js_rule_per_file"},
		{Path: "resolve/other/other1.js", Content: "goog.require('resolve.file1');"},
		{Path: "resolve/other/other2.js", Content: "goog.require('resolve.file2');"},

		// Test changing type of a rule as a jsx file is added.
		{
			Path: "change_type/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "change_type",
    srcs = ["file1.js"],
    visibility = ["//visibility:public"],
)
`,
		},
		{Path: "change_type/file1.js"},
		{Path: "change_type/file2.jsx"},

		// Test that activating rule_per_file works.
		{
			Path:    "rule_per_file/BUILD.bazel",
			Content: "# gazelle:js_rule_per_file true",
		},
		{Path: "rule_per_file/file1.js"},
		{Path: "rule_per_file/file2.js"},
		{
			Path:    "rule_per_file/reverted/BUILD.bazel",
			Content: "# gazelle:js_rule_per_file false",
		},
		{Path: "rule_per_file/reverted/file1.js"},
		{Path: "rule_per_file/reverted/file2.js"},

		// Test deleting files from a lib rule.
		{
			Path: "deleting_files/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "deleting_files",
    srcs = ["file1.js"],
    visibility = ["//visibility:public"],
)
`,
		},

		// Test renaming files.
		{
			Path: "renaming_files/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "renaming_files",
    srcs = ["file1.js"],
    visibility = ["//visibility:public"],
)
`,
		},
		{Path: "renaming_files/file2.js"},
	}

	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	if err := runGazelle(t, dir, []string{}); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{Path: "no_js/BUILD.bazel"},

		{
			Path: "no_rules/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "no_rules",
    srcs = ["norules.js"],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "lib",
    srcs = ["toplevel.js"],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "default_vis/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

package(default_visibility = ["//a:__subpackages__"])

closure_js_library(
    name = "default_vis",
    srcs = ["defaultvis.js"],
)
`,
		},

		{
			Path: "no_rules/multiple_files/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "multiple_files",
    srcs = [
        "a.js",
        "b.js",
        "c.js",
    ],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "no_rules/multiple_files_jsx/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_jsx_library")

closure_jsx_library(
    name = "multiple_files_jsx",
    srcs = [
        "a.js",
        "b.jsx",
        "c.js",
    ],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "no_rules/combine_deps/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_jsx_library")

# gazelle:js_grep_extern React //js/externs:react
# gazelle:js_grep_extern $ //js/externs:jQuery
# gazelle:js_grep_extern _ //js/externs:lodash

closure_jsx_library(
    name = "combine_deps",
    srcs = [
        "a.js",
        "b.jsx",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "//js/externs:jQuery",
        "//js/externs:lodash",
        "//js/externs:react",
        "@io_bazel_rules_closure//closure/library/array",
        "@io_bazel_rules_closure//closure/library/string",
        "@io_bazel_rules_closure//third_party/closure/library/dojo/dom:query",
    ],
)
`,
		},

		{
			Path: "existing_rules/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "existing",
    srcs = [
        "file1.js",
        "file2.js",
    ],
    data = ["testdata"],
    visibility = ["//visibility:public"],
    deps = [
        ":testdep",  # keep
        "@io_bazel_rules_closure//closure/library/string",
        "@io_bazel_rules_closure//third_party/closure/library/dojo/dom:query",
    ],
)

closure_js_library(
    name = "existingfile3",
    srcs = ["file3.js"],
    visibility = ["//visibility:public"],
    deps = [
        ":existing",
        "@io_bazel_rules_closure//closure/library/array",
    ],
)

closure_js_library(
    name = "deleted_some_multifile",
    srcs = ["file4.js"],
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "globbed",
    srcs = glob(
        ["*.js"],
        exclude = [
            "file1.js",
            "file2.js",
            "file3.js",
        ],
    ),
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "existing_rules",
    srcs = [
        "file5.js",
        "file6.js",
    ],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "existing_lib/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "existing_lib",
    srcs = [
        "file1.js",
        "file2.js",
    ],
    visibility = ["//visibility:public"],
    deps = [
        "@io_bazel_rules_closure//closure/library/array",
        "@io_bazel_rules_closure//closure/library/string",
    ],
)
`,
		},

		{
			Path: "resolve/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "resolve",
    srcs = [
        "file1.js",
        "file2.js",
    ],
    visibility = ["//visibility:public"],
)
`,
		},
		{
			Path: "resolve/other/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

# gazelle:js_rule_per_file

closure_js_library(
    name = "other1",
    srcs = ["other1.js"],
    visibility = ["//visibility:public"],
    deps = ["//resolve"],
)

closure_js_library(
    name = "other2",
    srcs = ["other2.js"],
    visibility = ["//visibility:public"],
    deps = ["//resolve"],
)
`,
		},

		{
			Path: "change_type/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_jsx_library")

closure_jsx_library(
    name = "change_type",
    srcs = [
        "file1.js",
        "file2.jsx",
    ],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "rule_per_file/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

# gazelle:js_rule_per_file true

closure_js_library(
    name = "file1",
    srcs = ["file1.js"],
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "file2",
    srcs = ["file2.js"],
    visibility = ["//visibility:public"],
)
`,
		},
		{
			Path: "rule_per_file/reverted/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

# gazelle:js_rule_per_file false

closure_js_library(
    name = "reverted",
    srcs = [
        "file1.js",
        "file2.js",
    ],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "deleting_files/BUILD.bazel",
		},

		{
			Path: "renaming_files/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library")

closure_js_library(
    name = "renaming_files",
    srcs = ["file2.js"],
    visibility = ["//visibility:public"],
)
`,
		},
	})
}

func TestJSTestRulePerDirectory(t *testing.T) {
	files := []testtools.FileSpec{
		{Path: "WORKSPACE"},

		// Test that a package with no rules has files combined into a directory.
		{
			Path:    "no_rules/norules.js",
			Content: `goog.provide('no_rules')`,
		},
		{
			Path:    "no_rules/norules_test.js",
			Content: ``,
		},

		// Test that default visibility is respected.
		{
			Path:    "default_vis/BUILD.bazel",
			Content: `package(default_visibility = ["//a:__subpackages__"])`,
		}, {
			Path: "default_vis/defaultvis_test.js",
		},

		// Test that a package with multiple tests has them combined.
		{Path: "no_rules/multiple_files/a_test.js"},
		{Path: "no_rules/multiple_files/b_test.js"},
		{Path: "no_rules/multiple_files/c_test.js"},

		// Test that a rule that includes a jsx file uses closure_jsx_library.
		{Path: "no_rules/multiple_files_jsx/a_test.js"},
		{Path: "no_rules/multiple_files_jsx/b_test.jsx"},
		{Path: "no_rules/multiple_files_jsx/c_test.js"},

		// Test that a rule that includes a HTML file is broken out.
		{Path: "no_rules/multiple_files_html/a_test.js"},
		{Path: "no_rules/multiple_files_html/b_test.jsx"},
		{Path: "no_rules/multiple_files_html/b_test.html"},
		{Path: "no_rules/multiple_files_html/c_test.js"},

		// Test that a package with existing rules (but no directory-level rule)
		// maintains the existing ones and adds new files to a library-wide
		// rule.
		{Path: "existing_rules/file1.js", Content: "goog.provide('existing_rules.file1');"},
		{Path: "existing_rules/file2.js", Content: "goog.provide('existing_rules.file2');"},
		{Path: "existing_rules/file1_test.js",
			Content: "goog.require('goog.string');\ngoog.require('existing_rules.file1');"},
		{Path: "existing_rules/file2_test.js",
			Content: "goog.require('goog.dom.query');\ngoog.require('existing_rules.file2');"},
		{Path: "existing_rules/file3_test.js"},
		{Path: "existing_rules/file4_test.jsx"},
		{Path: "existing_rules/file5_test.js"},
		{
			Path: "existing_rules/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_test")

closure_js_library(
    name = "existing_rules",
    srcs = ["file1.js"]
)

closure_js_library(
    name = "file2",
    srcs = ["file2.js"],
)

closure_js_test(
    name = "existing",
    srcs = ["file1_test.js", "file2_test.js"],
)

closure_js_test(
    name = "deleted",
    srcs = ["deleted.js"],
)

closure_js_test(
    name = "deleted_multifile",
    srcs = ["deleted1.js", "deleted2.js"],
)

closure_js_test(
    name = "deleted_some_multifile",
    srcs = ["file3_test.js", "deleted3.js"],
)
`,
		},

		// Test that a package with an existing directory-level rule adds to it.
		// Test that dependencies of individual files are combined.
		{
			Path: "existing_lib/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_test")

closure_js_library(
    name = "existing_lib",
    srcs = ["file1.js"],
)

closure_js_test(
    name = "existing_lib_test",
    srcs = ["file1_test.js"],
    visibility = ["//visibility:public"],
)
`,
		},
		{Path: "existing_lib/file1.js", Content: "goog.provide('existing_lib.file1');"},
		{Path: "existing_lib/file1_test.js", Content: "goog.require('goog.array');"},
		{Path: "existing_lib/file2_test.js",
			Content: "goog.provide('existing_lib.file2_test');\ngoog.require('existing_lib.file1');"},

		// Test changing type of a rule as a jsx file is added.
		{
			Path: "change_type/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_test")

closure_js_test(
    name = "change_type_test",
    srcs = ["file1_test.js"],
    visibility = ["//visibility:public"],
)
`,
		},
		{Path: "change_type/file1_test.js"},
		{Path: "change_type/file2_test.jsx"},

		// Test moving a test out of the directory-level rule if it gains a HTML.
		{
			Path: "gains_html/BUILD.bazel",
			Content: `
		load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_test")

		closure_js_test(
		    name = "gains_html_test",
		    srcs = ["a_test.js", "b_test.js"],
		    visibility = ["//visibility:public"],
		)
		`,
		},
		{Path: "gains_html/a_test.js"},
		{Path: "gains_html/b_test.js"},
		{Path: "gains_html/b_test.html"},

		// Test how a file with same name as directory and HTML is handled.
		//
		// Normally it would be broken out as a single-file rule, but the name
		// would conflict.
		{Path: "dir_has_html/dir_has_html_test.js", Content: "goog.require('goog.string');"},
		{Path: "dir_has_html/dir_has_html_test.html"},

		// Test handling a file gaining HTML when named the same as the directory.
		{
			Path: "gains_html_2/BUILD.bazel",
			Content: `
		load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_test")

		closure_js_test(
		    name = "gains_html_2_test",
		    srcs = ["gains_html_2_test.js", "b_test.js"],
		    visibility = ["//visibility:public"],
		)
		`,
		},
		{Path: "gains_html_2/gains_html_2_test.js", Content: `goog.require('goog.array');`},
		{Path: "gains_html_2/gains_html_2_test.html"},
		{Path: "gains_html_2/b_test.js", Content: `goog.require('goog.string');`},

		// Test a single rule whose source uses a different type.
		{
			Path: "change_type_2/BUILD.bazel",
			Content: `
        # gazelle:map_kind closure_jsx_test closure_jsx_test //tools/js:defs.bzl

		closure_js_test(
		    name = "a_test",
		    srcs = ["a_test.jsx"],
            data = ["testdata"],
		)

		closure_jsx_test(
		    name = "b_test",
		    srcs = ["b_test.js"],
            data = ["testdata"],
		)

		closure_jsx_test(
		    name = "c_test",
		    srcs = ["c_test.js"],
            data = ["testdata"],
		)

		closure_js_test(
		    name = "d_test",
		    srcs = ["d_test.jsx"],
            data = ["testdata"],
		)
`,
		},
		{Path: "change_type_2/a_test.jsx"},
		{Path: "change_type_2/a_test.html"},
		{Path: "change_type_2/b_test.js"},
		{Path: "change_type_2/b_test.html"},
		{Path: "change_type_2/c_test.js"},
		{Path: "change_type_2/d_test.jsx"},
		{Path: "change_type_2/testdata"},
	}

	dir, cleanup := testtools.CreateFiles(t, files)
	defer cleanup()

	if err := runGazelle(t, dir, []string{}); err != nil {
		t.Fatal(err)
	}

	testtools.CheckFiles(t, dir, []testtools.FileSpec{
		{
			Path: "no_rules/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library", "closure_js_test")

closure_js_library(
    name = "no_rules",
    srcs = ["norules.js"],
    visibility = ["//visibility:public"],
)

closure_js_test(
    name = "no_rules_test",
    srcs = ["norules_test.js"],
    compilation_level = "ADVANCED",
    entry_points = ["/no_rules/norules_test"],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "default_vis/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_test")

package(default_visibility = ["//a:__subpackages__"])

closure_js_test(
    name = "default_vis_test",
    srcs = ["defaultvis_test.js"],
    compilation_level = "ADVANCED",
    entry_points = ["/default_vis/defaultvis_test"],
)
`,
		},

		{
			Path: "no_rules/multiple_files/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_test")

closure_js_test(
    name = "multiple_files_test",
    srcs = [
        "a_test.js",
        "b_test.js",
        "c_test.js",
    ],
    compilation_level = "ADVANCED",
    entry_points = [
        "/no_rules/multiple_files/a_test",
        "/no_rules/multiple_files/b_test",
        "/no_rules/multiple_files/c_test",
    ],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "no_rules/multiple_files_jsx/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_jsx_test")

closure_jsx_test(
    name = "multiple_files_jsx_test",
    srcs = [
        "a_test.js",
        "b_test.jsx",
        "c_test.js",
    ],
    compilation_level = "ADVANCED",
    entry_points = [
        "/no_rules/multiple_files_jsx/a_test",
        "/no_rules/multiple_files_jsx/b_test",
        "/no_rules/multiple_files_jsx/c_test",
    ],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "no_rules/multiple_files_html/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_test", "closure_jsx_test")

closure_jsx_test(
    name = "b_test",
    srcs = ["b_test.jsx"],
    compilation_level = "ADVANCED",
    entry_points = ["/no_rules/multiple_files_html/b_test"],
    html = "b_test.html",
    visibility = ["//visibility:public"],
)

closure_js_test(
    name = "multiple_files_html_test",
    srcs = [
        "a_test.js",
        "c_test.js",
    ],
    compilation_level = "ADVANCED",
    entry_points = [
        "/no_rules/multiple_files_html/a_test",
        "/no_rules/multiple_files_html/c_test",
    ],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "existing_rules/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library", "closure_js_test", "closure_jsx_test")

closure_js_library(
    name = "existing_rules",
    srcs = ["file1.js"],
    visibility = ["//visibility:public"],
)

closure_js_library(
    name = "file2",
    srcs = ["file2.js"],
    visibility = ["//visibility:public"],
)

closure_js_test(
    name = "existing",
    srcs = [
        "file1_test.js",
        "file2_test.js",
    ],
    compilation_level = "ADVANCED",
    entry_points = [
        "/existing_rules/file1_test",
        "/existing_rules/file2_test",
    ],
    visibility = ["//visibility:public"],
    deps = [
        ":existing_rules",
        ":file2",
        "@io_bazel_rules_closure//closure/library/string",
        "@io_bazel_rules_closure//third_party/closure/library/dojo/dom:query",
    ],
)

closure_js_test(
    name = "deleted_some_multifile",
    srcs = ["file3_test.js"],
    compilation_level = "ADVANCED",
    entry_points = ["/existing_rules/file3_test"],
    visibility = ["//visibility:public"],
)

closure_jsx_test(
    name = "existing_rules_test",
    srcs = [
        "file4_test.jsx",
        "file5_test.js",
    ],
    compilation_level = "ADVANCED",
    entry_points = [
        "/existing_rules/file4_test",
        "/existing_rules/file5_test",
    ],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "existing_lib/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_library", "closure_js_test")

closure_js_library(
    name = "existing_lib",
    srcs = ["file1.js"],
    visibility = ["//visibility:public"],
)

closure_js_test(
    name = "existing_lib_test",
    srcs = [
        "file1_test.js",
        "file2_test.js",
    ],
    compilation_level = "ADVANCED",
    entry_points = [
        "/existing_lib/file1_test",
        "existing_lib.file2_test",
    ],
    visibility = ["//visibility:public"],
    deps = [
        ":existing_lib",
        "@io_bazel_rules_closure//closure/library/array",
    ],
)
`,
		},

		{
			Path: "change_type/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_jsx_test")

closure_jsx_test(
    name = "change_type_test",
    srcs = [
        "file1_test.js",
        "file2_test.jsx",
    ],
    compilation_level = "ADVANCED",
    entry_points = [
        "/change_type/file1_test",
        "/change_type/file2_test",
    ],
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "gains_html/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_test")

closure_js_test(
    name = "gains_html_test",
    srcs = ["a_test.js"],
    compilation_level = "ADVANCED",
    entry_points = ["/gains_html/a_test"],
    visibility = ["//visibility:public"],
)

closure_js_test(
    name = "b_test",
    srcs = ["b_test.js"],
    compilation_level = "ADVANCED",
    entry_points = ["/gains_html/b_test"],
    html = "b_test.html",
    visibility = ["//visibility:public"],
)
`,
		},

		{
			Path: "dir_has_html/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_test")

closure_js_test(
    name = "dir_has_html_test",
    srcs = ["dir_has_html_test.js"],
    compilation_level = "ADVANCED",
    entry_points = ["/dir_has_html/dir_has_html_test"],
    html = "dir_has_html_test.html",
    visibility = ["//visibility:public"],
    deps = ["@io_bazel_rules_closure//closure/library/string"],
)
`,
		},

		{
			Path: "gains_html_2/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_test")

closure_js_test(
    name = "gains_html_2_test",
    srcs = [
        "b_test.js",
        "gains_html_2_test.js",
    ],
    compilation_level = "ADVANCED",
    entry_points = [
        "/gains_html_2/b_test",
        "/gains_html_2/gains_html_2_test",
    ],
    html = "gains_html_2_test.html",
    visibility = ["//visibility:public"],
    deps = [
        "@io_bazel_rules_closure//closure/library/array",
        "@io_bazel_rules_closure//closure/library/string",
    ],
)
`,
		},

		{
			Path: "change_type_2/BUILD.bazel",
			Content: `
load("@io_bazel_rules_closure//closure:defs.bzl", "closure_js_test")
load("//tools/js:defs.bzl", "closure_jsx_test")

# gazelle:map_kind closure_jsx_test closure_jsx_test //tools/js:defs.bzl

closure_jsx_test(
    name = "a_test",
    srcs = ["a_test.jsx"],
    compilation_level = "ADVANCED",
    data = ["testdata"],
    entry_points = ["/change_type_2/a_test"],
    html = "a_test.html",
    visibility = ["//visibility:public"],
)

closure_js_test(
    name = "b_test",
    srcs = ["b_test.js"],
    compilation_level = "ADVANCED",
    data = ["testdata"],
    entry_points = ["/change_type_2/b_test"],
    html = "b_test.html",
    visibility = ["//visibility:public"],
)

closure_js_test(
    name = "c_test",
    srcs = ["c_test.js"],
    compilation_level = "ADVANCED",
    data = ["testdata"],
    entry_points = ["/change_type_2/c_test"],
    visibility = ["//visibility:public"],
)

closure_jsx_test(
    name = "d_test",
    srcs = ["d_test.jsx"],
    compilation_level = "ADVANCED",
    data = ["testdata"],
    entry_points = ["/change_type_2/d_test"],
    visibility = ["//visibility:public"],
)

`,
		},
	})
}

var gazellePath = flag.String("gazelle", "", "path to gazelle binary under test")

func runGazelle(t *testing.T, wd string, args []string) error {
	if *gazellePath == "" {
		t.Logf("-gazelle not specified, skipping")
		t.Skip()
	}
	t.Logf("running: %s", *gazellePath)

	oldWd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(wd); err != nil {
		return err
	}
	defer os.Chdir(oldWd)

	cmd := exec.Command(path.Join(oldWd, *gazellePath), args...)
	b, err := cmd.CombinedOutput()
	t.Logf("%s", b)
	return err
}
