package main

import (
	"encoding/csv"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}

	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close stdout pipe: %v", err)
	}
	os.Stdout = oldStdout

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read stdout pipe: %v", err)
	}
	return string(out)
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "field",
			path: "name",
			want: []string{"name"},
		},
		{
			name: "nested array field",
			path: "users[0].email",
			want: []string{"users", "[0]", "email"},
		},
		{
			name: "root array item",
			path: "[2]",
			want: []string{"[2]"},
		},
		{
			name: "leading dot",
			path: ".users[10].roles[1]",
			want: []string{"users", "[10]", "roles", "[1]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePath(tt.path)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parsePath(%q) = %#v, want %#v", tt.path, got, tt.want)
			}
		})
	}
}

func TestNavigate(t *testing.T) {
	data := map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"name":   "Jane",
				"active": true,
			},
		},
		"count": float64(1),
	}

	tests := []struct {
		name    string
		parts   []string
		want    interface{}
		wantErr string
	}{
		{
			name:  "nested array object value",
			parts: []string{"users", "[0]", "name"},
			want:  "Jane",
		},
		{
			name:  "root value",
			parts: []string{"count"},
			want:  float64(1),
		},
		{
			name:    "missing key",
			parts:   []string{"missing"},
			wantErr: "key not found: missing",
		},
		{
			name:    "array index out of range",
			parts:   []string{"users", "[1]"},
			wantErr: "index out of bounds: 1",
		},
		{
			name:    "field access on array",
			parts:   []string{"users", "name"},
			wantErr: "not an object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := navigate(data, tt.parts)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("navigate() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("navigate() returned unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("navigate() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestRunFmt(t *testing.T) {
	out := captureStdout(t, func() {
		runFmt([]byte(`{"b":2,"a":1}`))
	})

	want := "{\n  \"a\": 1,\n  \"b\": 2\n}\n"
	if out != want {
		t.Fatalf("runFmt() output = %q, want %q", out, want)
	}
}

func TestRunGet(t *testing.T) {
	input := []byte(`{"users":[{"name":"Jane","age":37}],"count":2}`)

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "string value",
			path: ".users[0].name",
			want: "Jane\n",
		},
		{
			name: "integer value",
			path: ".count",
			want: "2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, func() {
				runGet(input, tt.path)
			})
			if out != tt.want {
				t.Fatalf("runGet() output = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestRunKeysAndValues(t *testing.T) {
	keysOut := captureStdout(t, func() {
		runKeys([]byte(`{"name":"Jane","age":37}`))
	})

	gotKeys := strings.Fields(keysOut)
	if len(gotKeys) != 2 {
		t.Fatalf("runKeys() emitted %q, want two keys", keysOut)
	}
	keySet := map[string]bool{}
	for _, key := range gotKeys {
		keySet[key] = true
	}
	if !keySet["name"] || !keySet["age"] {
		t.Fatalf("runKeys() emitted %#v, want name and age", gotKeys)
	}

	valueOut := captureStdout(t, func() {
		runValues([]byte(`{"name":"Jane","age":37}`), "age")
	})
	if valueOut != "37\n" {
		t.Fatalf("runValues() output = %q, want 37", valueOut)
	}
}

func TestRunCountPluckAndLines(t *testing.T) {
	arrayInput := []byte(`[{"email":"jane@example.com"},{"email":"tarzan@example.com"},{"name":"Ignored"}]`)

	countOut := captureStdout(t, func() {
		runCount(arrayInput)
	})
	if countOut != "3\n" {
		t.Fatalf("runCount() output = %q, want 3", countOut)
	}

	pluckOut := captureStdout(t, func() {
		runPluck(arrayInput, "email")
	})
	if pluckOut != "jane@example.com\ntarzan@example.com\n" {
		t.Fatalf("runPluck() output = %q", pluckOut)
	}

	linesOut := captureStdout(t, func() {
		runLines([]byte(`[1,{"a":2},true]`))
	})
	if linesOut != "1\n{\"a\":2}\ntrue\n" {
		t.Fatalf("runLines() output = %q", linesOut)
	}
}

func TestRunCSV(t *testing.T) {
	out := captureStdout(t, func() {
		runCSV([]byte(`[{"name":"Jane","age":37},{"name":"Tarzan","age":41}]`))
	})

	records, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV output: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("CSV record count = %d, want 3", len(records))
	}

	rowAsMap := func(row []string) map[string]string {
		got := make(map[string]string, len(records[0]))
		for i, key := range records[0] {
			got[key] = row[i]
		}
		return got
	}

	if got := rowAsMap(records[1]); !reflect.DeepEqual(got, map[string]string{"name": "Jane", "age": "37"}) {
		t.Fatalf("first CSV row = %#v", got)
	}
	if got := rowAsMap(records[2]); !reflect.DeepEqual(got, map[string]string{"name": "Tarzan", "age": "41"}) {
		t.Fatalf("second CSV row = %#v", got)
	}
}

func TestRunValidate(t *testing.T) {
	out := captureStdout(t, func() {
		runValidate([]byte(`{"valid":true}`))
	})

	if out != "Valid JSON\n" {
		t.Fatalf("runValidate() output = %q, want valid message", out)
	}
}
