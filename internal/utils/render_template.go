package utils

import (
	"bytes"
	"html/template"
	"net/http"
)

// RenderTemplate safely executes a template into a buffer first.
// If it succeeds, it writes to the response. If it fails, it returns the error
// so the handler can issue an HTTP 500 without sending a malformed 200 OK body.
func RenderTemplate(
	w http.ResponseWriter,
	t *template.Template,
	name string,
	data any,
) error {
	var buf bytes.Buffer
	// Execute into the buffer
	if err := t.ExecuteTemplate(&buf, name, data); err != nil {
		return err
	}
	// If successful, write the buffer to the HTTP response
	_, err := buf.WriteTo(w)
	return err
}
