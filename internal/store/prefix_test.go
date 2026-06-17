package store

import (
	"reflect"
	"testing"
)

func TestParsePrefix(t *testing.T) {
	cases := []struct {
		in      string
		prefix  int
		slug    string
		ok      bool
	}{
		{"010-foo", 10, "foo", true},
		{"001-bar-baz", 1, "bar-baz", true},
		{"123-x", 123, "x", true},
		{"no-prefix", 0, "", false},
		{"01-too-short", 0, "", false},
		{"abcd-nope", 0, "", false},
	}
	for _, c := range cases {
		p, s, ok := ParsePrefix(c.in)
		if p != c.prefix || s != c.slug || ok != c.ok {
			t.Errorf("ParsePrefix(%q) = (%d, %q, %v), want (%d, %q, %v)",
				c.in, p, s, ok, c.prefix, c.slug, c.ok)
		}
	}
}

func TestFormatPrefix(t *testing.T) {
	if got := FormatPrefix(15, "foo"); got != "015-foo" {
		t.Errorf("got %q want 015-foo", got)
	}
	if got := FormatPrefix(120, "bar"); got != "120-bar" {
		t.Errorf("got %q want 120-bar", got)
	}
}

func TestPrefixBetween(t *testing.T) {
	cases := []struct {
		a, b   int
		want   int
		ok     bool
	}{
		{10, 20, 15, true},
		{10, 11, 0, false},
		{10, 12, 11, true},
		{20, 10, 15, true}, // order shouldn't matter
	}
	for _, c := range cases {
		got, ok := PrefixBetween(c.a, c.b)
		if got != c.want || ok != c.ok {
			t.Errorf("PrefixBetween(%d,%d) = (%d,%v), want (%d,%v)",
				c.a, c.b, got, ok, c.want, c.ok)
		}
	}
}

func TestRenumberGapOfTen(t *testing.T) {
	got := RenumberGapOfTen(3)
	want := []int{10, 20, 30}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}
