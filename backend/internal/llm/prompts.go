package llm

import (
	"bytes"
	"fmt"
	"text/template"
)

// Generate the templates logic

var promptTemplates *template.Template

// Initialize templates
func init() {
	promptTemplates = template.Must(template.New("").ParseGlob("internal/llm/prompts/*.tmpl"))
}

func RenderPrompt(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := promptTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("failed to render prompt template %s: %w", name, err)
	}
	return buf.String(), nil
}
