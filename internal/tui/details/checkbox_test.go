package details

import "testing"

func TestCountCheckboxes(t *testing.T) {
	body := "intro\n- [ ] a\n  - [x] b\nmiddle\n* [X] c\n- not a box"
	if n := CountCheckboxes(body); n != 3 {
		t.Errorf("count = %d, want 3", n)
	}
}

func TestToggleCheckboxFlips(t *testing.T) {
	body := "- [ ] a\n- [x] b\n- [ ] c"
	out, ok := ToggleCheckbox(body, 2)
	if !ok {
		t.Fatal("label 2 should toggle")
	}
	want := "- [ ] a\n- [ ] b\n- [ ] c"
	if out != want {
		t.Errorf("got %q\nwant %q", out, want)
	}
	out2, _ := ToggleCheckbox(out, 1)
	if out2 != "- [x] a\n- [ ] b\n- [ ] c" {
		t.Errorf("after toggling 1: %q", out2)
	}
}

func TestToggleCheckboxOutOfRange(t *testing.T) {
	body := "- [ ] a"
	if _, ok := ToggleCheckbox(body, 5); ok {
		t.Error("out-of-range label should return ok=false")
	}
	if _, ok := ToggleCheckbox(body, 0); ok {
		t.Error("label 0 should return ok=false")
	}
}

func TestToggleCheckboxPreservesIndent(t *testing.T) {
	body := "    - [ ] indented"
	out, ok := ToggleCheckbox(body, 1)
	if !ok {
		t.Fatal("toggle failed")
	}
	if out != "    - [x] indented" {
		t.Errorf("indent not preserved: %q", out)
	}
}
