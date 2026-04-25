package commands

import "testing"

func TestNormalizePageParity(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "1"},
		{"1", "1"},
		{"2", "2"},
		{"3", "3"},
		{" 2 ", "2"},
		{"0", "1"},
		{"4", "1"},
		{"abc", "1"},
	}
	for _, tc := range cases {
		if got := normalizePage(tc.in); got != tc.want {
			t.Fatalf("normalizePage(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeResParity(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "0"},
		{"0", "0"},
		{"1", "1"},
		{" 5 ", "5"},
		{"abc", "0"},
		{"-1", "0"},
	}
	for _, tc := range cases {
		if got := normalizeRes(tc.in); got != tc.want {
			t.Fatalf("normalizeRes(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
