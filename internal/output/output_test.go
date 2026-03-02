package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		expected string
	}{
		{
			name:     "slice",
			data:     []string{"a", "b"},
			expected: `["a","b"]`,
		},
		{
			name:     "struct",
			data:     struct{ Name string }{Name: "test"},
			expected: `{"Name":"test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := JSON(&buf, tt.data); err != nil {
				t.Fatalf("JSON() error: %v", err)
			}
			var want, got any
			if err := json.Unmarshal([]byte(tt.expected), &want); err != nil {
				t.Fatalf("unmarshal expected: %v", err)
			}
			if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal output: %v", err)
			}
			if fmt.Sprint(want) != fmt.Sprint(got) {
				t.Errorf("JSON() = %s, want %s", buf.String(), tt.expected)
			}
		})
	}
}

func TestTable(t *testing.T) {
	var buf bytes.Buffer
	err := Table(&buf, []string{"Name", "Value"}, [][]string{
		{"foo", "bar"},
		{"baz", "qux"},
	})
	if err != nil {
		t.Fatalf("Table() error: %v", err)
	}
	out := buf.String()
	upper := strings.ToUpper(out)
	for _, want := range []string{"NAME", "VALUE", "FOO", "BAR", "BAZ", "QUX"} {
		if !strings.Contains(upper, want) {
			t.Errorf("Table() output missing %q:\n%s", want, out)
		}
	}
}
