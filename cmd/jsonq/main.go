package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func usage() {
	fmt.Print(`jsonq — Simple JSON query tool

Usage: jsonq [command] [options] [file]

Commands:
  fmt          Pretty-print JSON
  get <path>   Get value by path (dot notation)
  keys         List top-level keys
  values       List values for a key
  count        Count array items
  pluck <k>    Extract field from array of objects
  csv          Convert array of objects to CSV
  validate     Check if valid JSON
  lines        Output array items one per line

Path syntax:
  .name         Object field
  .users[0]     Array index
  .users[0].id  Nested path

Examples:
  cat data.json | jsonq fmt
  jsonq get .users[0].name data.json
  jsonq pluck email users.json
  jsonq csv products.json > out.csv
  echo '{"a":1}' | jsonq validate
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var input []byte
	var err error

	// Read from file or stdin
	if len(args) > 0 && !strings.HasPrefix(args[len(args)-1], "-") {
		// Last arg might be a file
		last := args[len(args)-1]
		if _, err := os.Stat(last); err == nil {
			input, err = os.ReadFile(last)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error reading file:", err)
				os.Exit(1)
			}
			args = args[:len(args)-1]
		} else {
			input, err = io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error reading stdin:", err)
				os.Exit(1)
			}
		}
	} else {
		input, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading stdin:", err)
			os.Exit(1)
		}
	}

	if len(input) == 0 {
		fmt.Fprintln(os.Stderr, "No input provided")
		os.Exit(1)
	}

	switch cmd {
	case "fmt", "format":
		runFmt(input)
	case "get":
		if len(args) < 1 {
			fmt.Println("Usage: jsonq get <path> [file]")
			os.Exit(1)
		}
		runGet(input, args[0])
	case "keys":
		runKeys(input)
	case "values":
		if len(args) < 1 {
			fmt.Println("Usage: jsonq values <key> [file]")
			os.Exit(1)
		}
		runValues(input, args[0])
	case "count":
		runCount(input)
	case "pluck":
		if len(args) < 1 {
			fmt.Println("Usage: jsonq pluck <key> [file]")
			os.Exit(1)
		}
		runPluck(input, args[0])
	case "csv":
		runCSV(input)
	case "validate", "valid":
		runValidate(input)
	case "lines":
		runLines(input)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		usage()
		os.Exit(1)
	}
}

func runFmt(input []byte) {
	var v interface{}
	if err := json.Unmarshal(input, &v); err != nil {
		fmt.Fprintln(os.Stderr, "Invalid JSON:", err)
		os.Exit(1)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}

func runGet(input []byte, path string) {
	var v interface{}
	if err := json.Unmarshal(input, &v); err != nil {
		fmt.Fprintln(os.Stderr, "Invalid JSON:", err)
		os.Exit(1)
	}

	// Remove leading dot
	path = strings.TrimPrefix(path, ".")
	parts := parsePath(path)

	result, err := navigate(v, parts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	printValue(result)
}

func parsePath(path string) []string {
	var parts []string
	var current strings.Builder
	inBracket := false

	for _, r := range path {
		switch r {
		case '.':
			if !inBracket && current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		case '[':
			if !inBracket && current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			inBracket = true
		case ']':
			inBracket = false
			if current.Len() > 0 {
				parts = append(parts, "["+current.String()+"]")
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func navigate(v interface{}, parts []string) (interface{}, error) {
	if len(parts) == 0 {
		return v, nil
	}

	part := parts[0]
	rest := parts[1:]

	// Array index
	if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
		idxStr := part[1 : len(part)-1]
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			return nil, fmt.Errorf("invalid index: %s", idxStr)
		}
		arr, ok := v.([]interface{})
		if !ok {
			return nil, fmt.Errorf("not an array")
		}
		if idx < 0 || idx >= len(arr) {
			return nil, fmt.Errorf("index out of bounds: %d", idx)
		}
		return navigate(arr[idx], rest)
	}

	// Object field
	obj, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("not an object")
	}
	val, exists := obj[part]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", part)
	}
	return navigate(val, rest)
}

func printValue(v interface{}) {
	switch val := v.(type) {
	case string:
		fmt.Println(val)
	case float64:
		if val == float64(int64(val)) {
			fmt.Printf("%.0f\n", val)
		} else {
			fmt.Println(val)
		}
	case bool:
		fmt.Println(val)
	case nil:
		fmt.Println("null")
	default:
		out, _ := json.MarshalIndent(v, "", "  ")
		fmt.Println(string(out))
	}
}

func runKeys(input []byte) {
	var obj map[string]interface{}
	if err := json.Unmarshal(input, &obj); err != nil {
		fmt.Fprintln(os.Stderr, "Invalid JSON object:", err)
		os.Exit(1)
	}
	for k := range obj {
		fmt.Println(k)
	}
}

func runValues(input []byte, key string) {
	var obj map[string]interface{}
	if err := json.Unmarshal(input, &obj); err != nil {
		fmt.Fprintln(os.Stderr, "Invalid JSON object:", err)
		os.Exit(1)
	}
	val, exists := obj[key]
	if !exists {
		fmt.Fprintln(os.Stderr, "Key not found:", key)
		os.Exit(1)
	}
	printValue(val)
}

func runCount(input []byte) {
	var arr []interface{}
	if err := json.Unmarshal(input, &arr); err != nil {
		fmt.Fprintln(os.Stderr, "Invalid JSON array:", err)
		os.Exit(1)
	}
	fmt.Println(len(arr))
}

func runPluck(input []byte, key string) {
	var arr []interface{}
	if err := json.Unmarshal(input, &arr); err != nil {
		fmt.Fprintln(os.Stderr, "Invalid JSON array:", err)
		os.Exit(1)
	}
	for _, item := range arr {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if val, exists := obj[key]; exists {
			printValue(val)
		}
	}
}

func runCSV(input []byte) {
	var arr []map[string]interface{}
	if err := json.Unmarshal(input, &arr); err != nil {
		fmt.Fprintln(os.Stderr, "Invalid JSON array of objects:", err)
		os.Exit(1)
	}
	if len(arr) == 0 {
		fmt.Fprintln(os.Stderr, "Empty array")
		os.Exit(1)
	}

	// Collect all keys
	keySet := make(map[string]bool)
	for _, obj := range arr {
		for k := range obj {
			keySet[k] = true
		}
	}
	var keys []string
	for k := range keySet {
		keys = append(keys, k)
	}

	w := csv.NewWriter(os.Stdout)
	w.Write(keys)

	for _, obj := range arr {
		row := make([]string, len(keys))
		for i, k := range keys {
			if v, exists := obj[k]; exists {
				row[i] = fmt.Sprintf("%v", v)
			} else {
				row[i] = ""
			}
		}
		w.Write(row)
	}
	w.Flush()
}

func runValidate(input []byte) {
	var v interface{}
	if err := json.Unmarshal(input, &v); err != nil {
		fmt.Println("Invalid JSON:", err)
		os.Exit(1)
	}
	fmt.Println("Valid JSON")
}

func runLines(input []byte) {
	var arr []interface{}
	if err := json.Unmarshal(input, &arr); err != nil {
		fmt.Fprintln(os.Stderr, "Invalid JSON array:", err)
		os.Exit(1)
	}
	for _, item := range arr {
		out, _ := json.Marshal(item)
		fmt.Println(string(out))
	}
}
