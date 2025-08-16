package llm

import (
	"bytes"
	"fmt"
	txttmpl "text/template"
)

// SimpleTextInput represents a basic text input for embedding.
type SimpleTextInput struct {
	Content string
	Prefix  string
}

// NewSimpleTextInput creates a new SimpleTextInput with the given content.
func NewSimpleTextInput(content string) SimpleTextInput {
	return SimpleTextInput{Content: content}
}

// String returns the combined prefix and content of the SimpleTextInput.
func (t SimpleTextInput) String() string {
	return fmt.Sprintf("%s%s", t.Prefix, t.Content)
}

// EmbedInput defines the interface for types that can be used as input for embedding.
type EmbedInput interface {
	String() string
}

type PromptTemplateFactory struct {
	raw      string
	template *txttmpl.Template
}

func NewPromptTemplateFactory(template string) (*PromptTemplateFactory, error) {
	tmpl, err := txttmpl.New("query").Parse(template)
	if err != nil {
		return nil, err
	}

	return &PromptTemplateFactory{
		raw:      template,
		template: tmpl,
	}, nil
}

func (factory PromptTemplateFactory) NewPromptTemplate(vars map[string]any) *PromptTemplate {
	return &PromptTemplate{
		variables: vars,
		raw:       factory.raw,
		template:  factory.template,
	}
}


type PromptTemplate struct {
	variables map[string]any
	raw       string
	template  *txttmpl.Template
}

// NewPromptTemplate creates a new PromptTemplate with the given variables and template string.
func NewPromptTemplate(vars map[string]any, template string) (*PromptTemplate, error) {
	tmpl, err := txttmpl.New("query").Parse(template)
	if err != nil {
		return nil, err
	}

	return &PromptTemplate{
		variables: vars,
		raw:       template,
		template:  tmpl,
	}, nil
}

// Render executes the template with the stored variables and returns the resulting string.
func (q PromptTemplate) Render() (string, error) {
	buf := bytes.NewBuffer(nil)
	err := q.template.Execute(buf, q.variables)
	if err != nil {
		panic(err)
	}
	return buf.String(), nil
}

// String returns the rendered string of the PromptTemplate, or an error message if rendering fails.
func (q PromptTemplate) String() string {
	s, err := q.Render()
	if err != nil {
		return fmt.Sprintf("[template execution failed: %v]", err)
	}
	return s
}

// GetVar retrieves the value of a variable from the PromptTemplate.
func (q PromptTemplate) GetVar(key string) any {
	return q.variables[key]
}

// SetVar sets the value of a variable in the PromptTemplate.
func (q PromptTemplate) SetVar(key string, val any) {
	q.variables[key] = val
}

// Template returns the raw template string.
func (q PromptTemplate) Template() string {
	return q.raw
}

// InstructQuery creates a PromptTemplate with "instruct" and "query" variables.
func InstructQuery(instruct, query string) *PromptTemplate {
	pt, _ := NewPromptTemplate(
		map[string]any{
			"instruct": instruct,
			"query":    query,
		},
		"Instruct: {{.instruct}}\nQuery: {{.query}}",
	)
	return pt
}
