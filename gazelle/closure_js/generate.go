package closure_js

import (
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

type importInfo struct {
	imports, deps []string
}

func newImportInfo(fi fileInfo) importInfo {
	return importInfo{
		imports: fi.imports,
		deps:    fi.deps,
	}
}

func (gl *jsLang) GenerateRules(args language.GenerateArgs) language.GenerateResult {
	var jsc = getJsConfig(args.Config)

	// Visibility
	//
	// Calculation (same as in Go):
	// - If there's an internal/ ancestor directory, restrict visibility to
	//   rules declared within that directory's parent.
	// - Else, public.
	//
	// When generating a rule:
	// - If Default Visibility exists, do not set a visibility on the rule.
	// - Else, set visibility to (Calculation).
	var (
		hasDefaultVis bool
		visibility    string // empty means "don't set visibility"
	)
	if args.File != nil {
		hasDefaultVis = args.File.HasDefaultVisibility()
	}
	if !hasDefaultVis {
		visibility = rule.CheckInternalVisibility(args.Rel, "//visibility:public")
	}

	// Info about the directory-level library rule.
	var (
		libName         = path.Base(args.Rel)
		libExistingRule *rule.Rule
		libSources      []string
		libImports      []importInfo
	)
	if args.Rel == "" {
		libName = "lib" // top level of a workspace
	}

	// For each JS rule, extract info from the srcs and emit our take on it.
	// Keep track of the src files that were covered.
	var (
		rules            []*rule.Rule
		imports          []interface{}
		empty            []*rule.Rule
		existingRules    []*rule.Rule
		existingRuleSrcs []string
	)
	if args.File != nil {
		existingRules = args.File.Rules
	}

	var (
		libTestName         = libName + "_test"
		libTestFileInfos    []fileInfo
		libTestExistingRule *rule.Rule
	)

	// Index HTML so we're ready to provide them for test rules.
	var htmlLookup = make(map[string]fileInfo) // "a_test" => fileInfo{a_test.html}
	for _, filename := range args.RegularFiles {
		if path.Ext(filename) != ".html" {
			continue
		}
		var fi, ok = jsFileInfo(args.Config.RepoRoot, jsc, filepath.Join(args.Dir, filename))
		if !ok {
			continue
		}
		if fi.isTest && fi.ext == htmlExt {
			htmlLookup[noExt(fi.name)] = fi
			existingRuleSrcs = append(existingRuleSrcs, fi.name)
		}
	}

	for _, r := range existingRules {
		if !isJsLibrary(r.Kind()) && !isJsTest(r.Kind()) {
			continue
		}

		// A srcs attr with a glob appears as an empty srcs.
		// Just ignore those rules since there's nothing we can productively do.
		srcsattr := r.AttrStrings("srcs")
		if len(srcsattr) == 0 {
			continue
		}

		// Collect the combined srcs, requires, deps for this rule.
		var srcs, requires, deps []string
		var fileInfos []fileInfo
		for _, src := range srcsattr {
			// Ignore this src if it's a label.
			if strings.HasPrefix(src, ":") || strings.HasPrefix(src, "//") {
				srcs = append(srcs, src)
				continue
			}

			// Read the src and combine provides / imports.
			var fi, ok = jsFileInfo(args.Config.RepoRoot, jsc, filepath.Join(args.Dir, src))
			if !ok {
				continue
			}
			srcs = append(srcs, src)
			fileInfos = append(fileInfos, fi)

			switch fi.ext {
			case jsExt, jsxExt:
				deps = append(deps, fi.deps...)
				requires = append(requires, fi.imports...)
			}
		}
		existingRuleSrcs = append(existingRuleSrcs, srcs...)

		// If we're on the directory-level rule, record its sources and imports.
		// We can't emit a rule yet because we may need to add srcs.
		if !jsc.rulePerFile {
			switch r.Name() {
			case libName:
				libExistingRule = existingLib(r, srcs, visibility)
				libSources = append(libSources, srcs...)
				libImports = append(libImports, importInfo{
					imports: requires,
					deps:    deps,
				})
				continue
			case libTestName:
				libTestExistingRule = r
				libTestFileInfos = append(libTestFileInfos, fileInfos...)
				continue
			}
		}

		// Emit an empty rule if none of the srcs were present.
		// Else, emit the existing rule, possibly updated with our calculation
		// of its kind, imports, deps.
		if len(srcs) == 0 {
			empty = append(empty, existingLib(r, nil, visibility))
		} else if isJsTest(r.Kind()) {
			newRule := existingTest(r, fileInfos, htmlLookup[r.Name()], visibility)
			rules = append(rules,
				newRule)
			imports = append(imports, importInfo{
				imports: requires,
				deps:    dedupe(deps),
			})
			// If the kind changed, emit an empty rule under the old kind/name.
			if r.Kind() != newRule.Kind() {
				empty = append(empty, rule.NewRule(r.Kind(), r.Name()))
			}
		} else {
			rules = append(rules, existingLib(r, srcs, visibility))
			imports = append(imports, importInfo{
				imports: requires,
				deps:    dedupe(deps),
			})
		}
	}

	// Loop through each file in this package, read their info (what they
	// provide & require), and generate a lib or test rule for it.
	for _, filename := range args.RegularFiles {
		// Skip any that we already processed.
		if contains(existingRuleSrcs, filename) {
			continue
		}

		var fi, ok = jsFileInfo(args.Config.RepoRoot, jsc, filepath.Join(args.Dir, filename))
		if !ok {
			continue
		}
		if fi.ext == unknownExt {
			continue
		}

		// Deal with tests separately since they involve multiple files.
		if fi.isTest {
			switch fi.ext {
			case jsExt, jsxExt:
				libTestFileInfos = append(libTestFileInfos, fi)
			}
			continue
		}

		// Create one closure_js[x]_library rule per non-test source file.
		switch fi.ext {
		case jsExt, jsxExt:
			libSources = append(libSources, filename)
			libImports = append(libImports, newImportInfo(fi))
		}
	}

	// Generate libraries.
	// Group the libs together unless filePerRule is set.
	if jsc.rulePerFile {
		for i, src := range libSources {
			name := src[:len(src)-len(filepath.Ext(src))]
			rules = append(rules, generateLib(name, []string{src}, visibility))
			imports = append(imports, libImports[i])
		}
	} else if len(libImports) > 0 {
		if len(libSources) == 0 {
			empty = append(empty, libExistingRule)
		} else {
			generatedLib := generateLib(libName, libSources, visibility)
			rules = append(rules, generatedLib)
			imports = append(imports, combineImports(libImports))

			// If the existing and generated rule have different kinds
			// (closure_js_library vs closure_jsx_library), emit an empty rule
			// to delete the existing one, so our new one takes priority.
			if libExistingRule != nil &&
				libExistingRule.Kind() != generatedLib.Kind() {
				empty = append(empty, existingLib(libExistingRule, nil, visibility))
			}
		}
	}

	// Generate tests.
	if jsc.rulePerFile {
		for _, fi := range libTestFileInfos {
			if html, ok := htmlLookup[noExt(fi.name)]; ok {
				rules = append(rules, generateCombinedTest(fi, html, visibility))
			} else {
				rules = append(rules, generateTest(fi, visibility))
			}
			imports = append(imports, newImportInfo(fi))
		}
	} else {
		if len(libTestFileInfos) == 0 {
			empty = append(empty, generateMultiTest(libTestName, nil, visibility))
		} else {
			// Create individual rules for tests that have associated HTML.
			// Create a single directory level rule for all others.
			//
			// Special case: if individual test with associated HTML has same
			// name as the directory, set the html on it regardless.
			var noHtmlFileInfos []fileInfo
			var dirHTML fileInfo // only set for the special case above
			for _, fi := range libTestFileInfos {
				if html, ok := htmlLookup[noExt(fi.name)]; ok {
					if fi.name[:len(fi.name)-len(path.Ext(fi.name))] == libTestName {
						dirHTML = html
						noHtmlFileInfos = append(noHtmlFileInfos, fi)
						continue
					}
					rules = append(rules, generateCombinedTest(fi, html, visibility))
					imports = append(imports, fileInfoImports([]fileInfo{fi}))
				} else {
					noHtmlFileInfos = append(noHtmlFileInfos, fi)
				}
			}

			if len(noHtmlFileInfos) > 0 {
				generatedTest := generateMultiTest(libTestName, noHtmlFileInfos, visibility)
				if dirHTML.path != "" {
					generatedTest.SetAttr("html", dirHTML.name)
				}
				rules = append(rules, generatedTest)
				imports = append(imports, fileInfoImports(noHtmlFileInfos))

				// If the existing and generated rule have different kinds
				// (closure_js_test vs closure_jsx_test), emit an empty rule
				// to delete the existing one, so our new one takes priority.
				if libTestExistingRule != nil &&
					libTestExistingRule.Kind() != generatedTest.Kind() {
					empty = append(empty,
						rule.NewRule(libTestExistingRule.Kind(), libTestExistingRule.Name()))
				}
			}
		}
	}

	return language.GenerateResult{
		Gen:     rules,
		Imports: imports,
		Empty:   empty,
	}
}

func fileInfoImports(fis []fileInfo) importInfo {
	var ii importInfo
	for _, fi := range fis {
		ii.imports = append(ii.imports, fi.imports...)
		ii.deps = append(ii.deps, fi.deps...)
	}
	ii.deps = dedupe(ii.deps)
	return ii
}

// noExt trims foo_test.[js|html] => "foo_test"
func noExt(name string) string {
	return name[:len(name)-len(path.Ext(name))]
}

func existingLib(existing *rule.Rule, srcs []string, vis string) *rule.Rule {
	r := rule.NewRule(existing.Kind(), existing.Name())
	if len(srcs) > 0 {
		r.SetAttr("srcs", srcs)
	}
	if vis != "" {
		r.SetAttr("visibility", []string{vis})
	}
	return r
}

func generateLib(name string, srcs []string, vis string) *rule.Rule {
	jsOrJsx := "js"
	for _, src := range srcs {
		if filepath.Ext(src) == ".jsx" {
			jsOrJsx = "jsx"
			break
		}
	}
	r := rule.NewRule("closure_"+jsOrJsx+"_library", name)
	if len(srcs) > 0 {
		r.SetAttr("srcs", srcs)
	}
	if vis != "" {
		r.SetAttr("visibility", []string{vis})
	}
	return r
}

func existingTest(r *rule.Rule, js []fileInfo, html fileInfo, vis string) *rule.Rule {
	var srcs, provides []string
	var jsOrJsx = "js"
	for _, fi := range js {
		srcs = append(srcs, fi.name)
		provides = append(provides, fi.provides...)
		if filepath.Ext(fi.name) == ".jsx" {
			jsOrJsx = "jsx"
		}
	}
	sort.Strings(srcs)
	sort.Strings(provides)

	// Calculate and set srcs and entry_points.
	// All other attrs, defer to what's already there.
	newRule := rule.NewRule("closure_"+jsOrJsx+"_test", r.Name())
	for _, attrName := range r.AttrKeys() {
		if attrName != "srcs" && attrName != "entry_points" {
			newRule.SetAttr(attrName, r.Attr(attrName))
		}
	}
	if len(srcs) > 0 {
		newRule.SetAttr("srcs", srcs)
	}
	if len(provides) > 0 {
		newRule.SetAttr("entry_points", provides)
	}
	if r.Attr("compilation_level") == nil {
		newRule.SetAttr("compilation_level", "ADVANCED")
	}
	if r.Attr("visibility") == nil && vis != "" {
		newRule.SetAttr("visibility", []string{vis})
	}
	if r.Attr("html") == nil && html.path != "" {
		newRule.SetAttr("html", html.name)
	}
	return newRule
}

func generateMultiTest(name string, js []fileInfo, vis string) *rule.Rule {
	var srcs, provides []string
	var jsOrJsx = "js"
	for _, fi := range js {
		srcs = append(srcs, fi.name)
		provides = append(provides, fi.provides...)
		if filepath.Ext(fi.name) == ".jsx" {
			jsOrJsx = "jsx"
		}
	}
	sort.Strings(srcs)
	sort.Strings(provides)

	r := rule.NewRule("closure_"+jsOrJsx+"_test", name)
	if len(srcs) > 0 {
		r.SetAttr("srcs", srcs)
	}
	r.SetAttr("compilation_level", "ADVANCED")
	if len(provides) > 0 {
		r.SetAttr("entry_points", provides)
	}
	if vis != "" {
		r.SetAttr("visibility", []string{vis})
	}
	return r
}

func generateCombinedTest(js, html fileInfo, vis string) *rule.Rule {
	r := generateTest(js, vis)
	r.SetAttr("html", html.name)
	return r
}

func generateTest(js fileInfo, vis string) *rule.Rule {
	jsOrJsx := filepath.Ext(js.name)[1:]
	r := rule.NewRule("closure_"+jsOrJsx+"_test",
		js.name[:len(js.name)-len(filepath.Ext(js.name))])
	r.SetAttr("srcs", []string{js.name})
	r.SetAttr("compilation_level", "ADVANCED")
	r.SetAttr("entry_points", js.provides)
	if vis != "" {
		r.SetAttr("visibility", []string{vis})
	}
	return r
}

func combineImports(imports []importInfo) importInfo {
	var out importInfo
	for _, ii := range imports {
		out.deps = append(out.deps, ii.deps...)
		out.imports = append(out.imports, ii.imports...)
	}
	out.deps = dedupe(out.deps)
	return out
}

func dedupe(deps []string) []string {
	var out = make([]string, 0, len(deps))
	var dict = make(map[string]struct{})
	for _, dep := range deps {
		if _, ok := dict[dep]; ok {
			continue
		}
		dict[dep] = struct{}{}
		out = append(out, dep)
	}
	return out
}
