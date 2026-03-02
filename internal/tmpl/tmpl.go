package tmpl

import (
	"bytes"
	"fmt"
	"text/template"
)

// Resolve parses text as a Go text/template with the given name and executes
// it against data.
func Resolve(name, text string, data any) (string, error) {
	t, err := template.New(name).Parse(text)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}
