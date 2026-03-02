package tmpl

import (
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    string
		data    any
		want    string
		wantErr string
	}{
		{
			name: "simple substitution",
			tmpl: "hello {{.Name}}",
			data: map[string]string{"Name": "world"},
			want: "hello world",
		},
		{
			name: "empty template",
			tmpl: "",
			data: nil,
			want: "",
		},
		{
			name:    "parse error",
			tmpl:    "{{.bad",
			data:    nil,
			wantErr: "parse template",
		},
		{
			name:    "execute error with func call",
			tmpl:    "{{.Name.Bad}}",
			data:    map[string]string{"Name": "val"},
			wantErr: "execute template",
		},
		{
			name: "nested data",
			tmpl: "got: {{.step1.output}}",
			data: map[string]any{
				"step1": map[string]any{"output": "result"},
			},
			want: "got: result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve("test", tt.tmpl, tt.data)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
