package renderer

import (
	"html/template"
	"io"
	"log"

	"github.com/labstack/echo/v4"
)

// TemplateRenderer ...
type TemplateRenderer struct {
	templates *template.Template
}

// NewRenderer ...
func NewRenderer(templateDir string) *TemplateRenderer {
	templates, err := template.ParseGlob(templateDir + "/*.html")
	if err != nil {
		log.Fatalf("Ошибка загрузки шаблонов: %v", err)
	}
	return &TemplateRenderer{templates: templates}
}

// Render ...
func (t *TemplateRenderer) Render(
	w io.Writer,
	name string,
	data interface{},
	c echo.Context,
) error {
	if viewData, ok := data.(map[string]interface{}); ok {
		viewData["CSRFToken"] = c.Get("csrf")
		return t.templates.ExecuteTemplate(w, name, viewData)
	}
	return t.templates.ExecuteTemplate(w, name, data)
}
