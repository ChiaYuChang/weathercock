package llm

import "encoding/json"

type Role string

func (r Role) String() string {
	return string(r)
}

func (r Role) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}

const (
	RoleDeveloper Role = "developer"
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleFunction  Role = "function"
)

type BaseMessage struct {
	role    Role        `json:"-"`
	Name    string      `json:"name,omitempty"`
	content []InputText `json:"-"`
}

func NewBaseMessage(role Role, name string, content ...string) BaseMessage {
	text := make([]InputText, len(content))
	for i, c := range content {
		text[i] = NewInputText(c)
	}

	return BaseMessage{
		role:    role,
		Name:    name,
		content: text,
	}
}

func (msg *BaseMessage) Append(content ...string) *BaseMessage {
	for _, c := range content {
		msg.content = append(msg.content, NewInputText(c))
	}
	return msg
}

func (msg *BaseMessage) SetRole(role Role) *BaseMessage {
	msg.role = role
	return msg
}

func (msg BaseMessage) MarshalJSON() ([]byte, error) {
	if len(msg.content) == 1 {
		type Alias BaseMessage
		return json.Marshal(struct {
			Role    string `json:"role"`
			Content string `json:"content"`
			*Alias
		}{
			Role:    msg.role.String(),
			Content: msg.content[0].Text,
			Alias:   (*Alias)(&msg),
		})
	}

	type Alias BaseMessage
	return json.Marshal(struct {
		Role    string      `json:"role"`
		Content []InputText `json:"content"`
		*Alias
	}{
		Role:    msg.role.String(),
		Content: msg.content,
		Alias:   (*Alias)(&msg),
	})
}

func (msg BaseMessage) Role() string {
	return msg.role.String()
}

func (msg BaseMessage) Content() []string {
	content := make([]string, len(msg.content))
	for i, c := range msg.content {
		content[i] = c.Text
	}
	return content
}

type InputText struct {
	Text string `json:"text"`
}

func NewInputText(text string) InputText {
	return InputText{text}
}

func (mt InputText) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{
		Type: "text",
		Text: mt.Text,
	})
}

func NewDeveloperMessage(name string, content ...string) ChatCompletionMessage {
	return NewBaseMessage(RoleDeveloper, name, content...)
}

func NewSystemMessage(name string, content ...string) ChatCompletionMessage {
	return NewBaseMessage(RoleSystem, name, content...)
}

func NewUserMessage(name string, content ...string) ChatCompletionMessage {
	return NewBaseMessage(RoleUser, name, content...)
}

type ToolMessage struct {
	content    []InputText `json:"-"`
	ToolCallID string      `json:"tool_call_id"`
}

func NewToolMessage(tcID string, content ...string) ChatCompletionMessage {
	text := make([]InputText, len(content))
	for i, c := range content {
		text[i] = NewInputText(c)
	}

	return ToolMessage{
		content:    text,
		ToolCallID: tcID,
	}
}

func (msg ToolMessage) Role() string {
	return RoleFunction.String()
}

func (msg ToolMessage) Content() []string {
	content := make([]string, len(msg.content))
	for i, c := range msg.content {
		content[i] = c.Text
	}
	return content
}

func (msg ToolMessage) MarshalJSON() ([]byte, error) {
	if len(msg.content) == 1 {
		type Alias ToolMessage
		return json.Marshal(struct {
			Role    string `json:"role"`
			Content string `json:"content"`
			*Alias
		}{
			Role:    msg.Role(),
			Content: msg.content[0].Text,
			Alias:   (*Alias)(&msg),
		})
	}

	type Alias ToolMessage
	return json.Marshal(struct {
		Role    string      `json:"role"`
		Content []InputText `json:"content"`
		*Alias
	}{
		Role:    msg.Role(),
		Content: msg.content,
		Alias:   (*Alias)(&msg),
	})
}

type AssistantMessage struct {
	BaseMessage
	Audio struct {
		ID string `json:"id"`
	} `json:"audio,omitempty"`
	Refusal    string        `json:"refusal,omitempty"`
	ToolsCalls []ToolMessage `json:"tool_calls,omitempty"`
}

func NewAssistantMessage(name string, content ...string) ChatCompletionMessage {
	text := make([]InputText, len(content))
	for i, c := range content {
		text[i] = NewInputText(c)
	}
	return AssistantMessage{
		BaseMessage: BaseMessage{
			role:    RoleAssistant,
			Name:    name,
			content: text,
		},
		ToolsCalls: []ToolMessage{},
	}
}

func (msg AssistantMessage) MarshalJSON() ([]byte, error) {
	if len(msg.content) == 1 {
		type Alias AssistantMessage
		return json.Marshal(struct {
			Role    string `json:"role"`
			Content string `json:"content"`
			*Alias
		}{
			Role:    msg.role.String(),
			Content: msg.content[0].Text,
			Alias:   (*Alias)(&msg),
		})
	}

	type Alias AssistantMessage
	return json.Marshal(struct {
		Role    string      `json:"role"`
		Content []InputText `json:"content"`
		*Alias
	}{
		Role:    msg.role.String(),
		Content: msg.content,
		Alias:   (*Alias)(&msg),
	})
}

func (msg AssistantMessage) Role() string {
	return msg.role.String()
}

type ToolCall struct {
	ID       string           `json:"id"`
	Function ToolCallFunction `json:"function"`
}

func (tc ToolCall) MarshalJSON() ([]byte, error) {
	type Alias ToolCall
	return json.Marshal(struct {
		*Alias
		Type string `json:"type"`
	}{
		Alias: (*Alias)(&tc),
		Type:  "function",
	})
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
