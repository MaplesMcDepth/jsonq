package main

import (
	"bytes"
	"reflect"
	"testing"
)

func TestParsePath(t *testing.T) {
	got := parsePath("users[0].profile.name")
	want := []string{"users", "[0]", "profile", "name"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parsePath mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestNavigateNestedValue(t *testing.T) {
	input := map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"profile": map[string]interface{}{"name": "Ada"},
			},
		},
	}

	got, err := navigate(input, []string{"users", "[0]", "profile", "name"})
	if err != nil {
		t.Fatalf("navigate returned error: %v", err)
	}
	if got != "Ada" {
		t.Fatalf("navigate returned %v, want Ada", got)
	}
}

func TestWriteCSVSortsKeysAndPadsMissingValues(t *testing.T) {
	arr := []map[string]interface{}{
		{"b": 2, "a": 1},
		{"a": 3},
	}

	var buf bytes.Buffer
	if err := writeCSV(&buf, arr); err != nil {
		t.Fatalf("writeCSV returned error: %v", err)
	}

	want := "a,b\n1,2\n3,\n"
	if buf.String() != want {
		t.Fatalf("writeCSV output mismatch\n got: %q\nwant: %q", buf.String(), want)
	}
}
