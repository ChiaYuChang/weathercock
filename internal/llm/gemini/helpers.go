package gemini

import (
	"errors"
	"fmt"

	"github.com/ChiaYuChang/weathercock/internal/llm"
	"google.golang.org/genai"
)

var (
	ErrMassageConvertFailed = errors.New("could not convert messages to GenAI contents")
	ErrInvalidConfigType    = errors.New("invalid config type")
)

func isTerminalJobState(status string) bool {
	switch genai.JobState(status) {
	case genai.JobStateSucceeded,
		genai.JobStateFailed,
		genai.JobStateCancelled,
		genai.JobStateExpired:
		return true
	default:
		return false
	}
}

// toGenAIContents conver []llm.Messages to []*genai.Contetn
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
