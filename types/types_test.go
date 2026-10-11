package types

import "testing"

func TestPathLabelsIsEmpty(t *testing.T) {
	if !(PathLabels{}).IsEmpty() {
		t.Fatal("zero labels must be empty")
	}
	for _, labels := range []PathLabels{{Method: "m"}, {Param: "p"}, {Schema: "s"}, {Descriptor: "d"}, {Tag: "t"}, {Section: "info"}} {
		if labels.IsEmpty() {
			t.Fatalf("nonempty labels: %+v", labels)
		}
	}
}
