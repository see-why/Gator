package main

import "testing"

func TestParsePageArg_Valid(t *testing.T) {
	cases := map[string]int32{
		"1":  1,
		"5":  5,
		"10": 10,
	}
	for input, want := range cases {
		got, err := parsePageArg(input)
		if err != nil {
			t.Fatalf("parsePageArg(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("parsePageArg(%q) = %d; want %d", input, got, want)
		}
	}
}

func TestParsePageArg_Invalid(t *testing.T) {
	inputs := []string{"", "abc", "0", "-1", "1.5"}
	for _, input := range inputs {
		if _, err := parsePageArg(input); err == nil {
			t.Fatalf("expected error for input %q", input)
		}
	}
}
