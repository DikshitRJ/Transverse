package llm

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed prompts/*.tmpl
var promptsFS embed.FS

var promptTemplates = template.Must(template.ParseFS(promptsFS, "prompts/*.tmpl"))

func RenderPrompt(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := promptTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("failed to render prompt template %s: %w", name, err)
	}
	return buf.String(), nil
}
