package paths

import "testing"

func TestConcretePaths(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"$", "$"}, {"['info']", "$['info']"},
		{"$[''][0]['a\\'b']", "$[''][0]['a\\'b']"},
		{"unknown", ""}, {"$..name", ""}, {"$[*]", ""},
		{"$[0,1]", ""}, {"$[-1]", ""},
	} {
		location, valid := Parse(tc.input)
		if valid != (tc.want != "") {
			t.Fatalf("Parse(%q): valid=%v, want %v", tc.input, valid, tc.want != "")
		}
		if valid && location.Selectors().String() != tc.want {
			t.Fatalf("Parse(%q): %q, want %q", tc.input, location.Selectors().String(), tc.want)
		}
	}
}

func TestPathDisplayAndMethodOrder(t *testing.T) {
	for _, tc := range []struct {
		input, display string
		method         int
	}{
		{"$", "$", -1}, {"unknown", "unknown", -1},
		{"$['info']['license']", "info.license", -1},
		{"$[''][0]['a\\'b']", "[''][0].a'b", -1},
		{"$[0]['methods'][2]", "[0].methods[2]", 2},
		{"$['methods']['name']", "methods.name", -1},
		{"$['methods'][17]['result']", "methods[17].result", 17},
	} {
		path, _ := Parse(tc.input)
		if got := path.Friendly(); got != tc.display {
			t.Errorf("Friendly(%q) = %q, want %q", tc.input, got, tc.display)
		}
		if got := path.MethodIndex(); got != tc.method {
			t.Errorf("MethodIndex(%q) = %d, want %d", tc.input, got, tc.method)
		}
	}
}
