package main

import (
	"testing"
)

func TestParseSegment(t *testing.T) {
	tests := []struct {
		input         string
		expectedTerm  string
		expectedAlias string
	}{
		{"Waterloo as KW", "Waterloo", "KW"},
		{"London", "London", ""},
		{"Toronto  as  YYZ", "Toronto", "YYZ"},
	}

	for _, tt := range tests {
		term, alias := parseSegment(tt.input)
		if term != tt.expectedTerm || alias != tt.expectedAlias {
			t.Errorf("parseSegment(%q) = %q, %q; want %q, %q", tt.input, term, alias, tt.expectedTerm, tt.expectedAlias)
		}
	}
}

func TestParsePathSegments(t *testing.T) {
	input := "/Waterloo/Toronto/"
	expected := []string{"Waterloo", "Toronto"}
	
	result := parsePathSegments(input)
	if len(result) != len(expected) {
		t.Fatalf("parsePathSegments() returned %d items, want %d", len(result), len(expected))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("parsePathSegments()[%d] = %q, want %q", i, result[i], v)
		}
	}
}

func TestFindCity(t *testing.T) {
	city, ok := findCity("Waterloo")
	if !ok {
		t.Error("findCity(Waterloo) should be true")
	}
	if city.Name != "Waterloo" {
		t.Errorf("expected Waterloo, got %s", city.Name)
	}

	_, ok = findCity("NotARealCity123")
	if ok {
		t.Error("findCity(NotARealCity123) should be false")
	}
}
