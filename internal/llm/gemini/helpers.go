package gemini

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"google.golang.org/genai"
)

var (
	ErrMassageConvertFailed = errors.New("could not convert messages to GenAI contents")
	ErrInvalidConfigType    = errors.New("invalid config type")
)

// IsTerminalJobState checks if a given job status indicates a terminal state (succeeded, failed, cancelled, or expired).
func IsTerminalJobState(status genai.JobState) bool {
	switch status {
	case genai.JobStateSucceeded,
		genai.JobStateFailed,
		genai.JobStateCancelled,
		genai.JobStateExpired:
		return true
	default:
		return false
	}
}

// toGenAIContents converts a slice of llm.Message to a slice of *genai.Content.
func toGenAIContents(messages []llm.Message) ([]*genai.Content, error) {
	contents := make([]*genai.Content, len(messages))
	for i, msg := range messages {
		parts := make([]*genai.Part, len(msg.Content))
		for j, text := range msg.Content {
			parts[j] = genai.NewPartFromText(text)
		}

		var role genai.Role
		switch msg.Role {
		case llm.RoleUser:
			role = genai.RoleUser
		case llm.RoleSystem, llm.RoleAssistant:
			role = genai.RoleModel
		default:
			return nil, fmt.Errorf("%w: unsupported role: %s", ErrMassageConvertFailed, msg.Role)
		}
		contents[i] = genai.NewContentFromParts(parts, role)
	}
	return contents, nil
}

// assertAs performs a type assertion, returning the result or an error if the assertion fails.
func assertAs[T any](conf any) (T, error) {
	if conf == nil {
		var zero T
		return zero, nil
	}

	gConf, ok := conf.(T)
	if !ok {
		return gConf, fmt.Errorf("%w: %T, should be %T", ErrInvalidConfigType, conf, gConf)
	}
	return gConf, nil
}

func extractJSONObject(s string) (string, error) {
	start := strings.Index(s, "{")
	if start == -1 {
		return "", fmt.Errorf("could not find opening brace '{' in the string")
	}

	end := strings.LastIndex(s, "}")
	if end == -1 {
		return "", fmt.Errorf("could not find closing brace '}' in the string")
	}

	if end < start {
		return "", fmt.Errorf("found closing brace '}' before opening brace '{'")
	}

	// Slice the string from the first '{' to the last '}'
	return s[start : end+1], nil
}
