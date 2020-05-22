package closure_js

import "testing"

func TestResolveClosureLibrary(t *testing.T) {
	var tests = []struct {
		in, out string
	}{
		{"goog.i18n.MessageFormat", "@io_bazel_rules_closure//closure/library/i18n:messageformat"},
	}
	for _, test := range tests {
		actual := resolveClosureLibrary(test.in).String()
		if actual != test.out {
			t.Errorf("%v: expected %v, got %v", test.in, test.out, actual)
		}
	}
}
func TestIsJsLibrary(t *testing.T) {
	var tests = []struct {
		in  string
		out bool
	}{
		{"closure_js_library", true},
		{"closure_js_test", true},
		{"closure_jsx_library", true},
		{"closure_jsx_test", true},
		{"closure_js_thirdparty_library", true},
		{"closure_js_template_library", false},
		{"closure_jsx_template_library", false},
	}
	for _, test := range tests {
		actual := isJsLibrary(test.in)
		if actual != test.out {
			t.Errorf("%v: expected %v, got %v", test.in, test.out, actual)
		}
	}
}
