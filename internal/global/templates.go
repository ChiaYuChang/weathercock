package global

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/ChiaYuChang/weathercock/pkgs/utils"
	"github.com/spf13/viper"
)

// TemplateConfig holds directory and file pattern for templates.
type TemplateConfig struct {
	Dir  string `json:"dir"  validate:"required"`
	File string `json:"file" validate:"required"`
}

func (tmpl TemplateConfig) Path() string {
	return tmpl.Dir + "/" + tmpl.File
}

func (tmpl TemplateConfig) String() string {
	return fmt.Sprintf("TemplateConfig{Dir: %s, File: %s}", tmpl.Dir, tmpl.File)
}

func (tmpl TemplateConfig) Validate() error {
	return Validator.Struct(tmpl)
}

func TemplatesConfig() *TemplateConfig {
	return &TemplateConfig{
		Dir:  utils.DefaultIfZero(viper.GetString("TMPL_DIR"), "./templates"),
		File: utils.DefaultIfZero(viper.GetString("TMPL_FILE"), "*.gotmpl"),
	}
}

func TemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"timeFormat": func(t time.Time, layout string) string {
			return t.Format(layout)
		},
		"join": func(items []string, sep string) string {
			if len(items) == 0 {
				return ""
			}
			qs := make([]string, len(items))
			for i, item := range items {
				qs[i] = fmt.Sprintf("%q", item)
			}
			return strings.Join(qs, sep)
		},
		"hidden": func(s string) string {
			if len(s) <= 10 {
				return strings.Repeat("*", len(s))
			}
			return s[:5] + strings.Repeat("*", len(s)-10) + s[len(s)-5:]
		},
	}
}

func TemplateRepo(funcs template.FuncMap, pattern string) (*template.Template, error) {
	tmpl, err := template.New("").
		Funcs(funcs).
		ParseGlob(pattern)

	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return tmpl, nil
}
