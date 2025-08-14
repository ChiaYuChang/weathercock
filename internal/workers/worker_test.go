package workers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ChiaYuChang/weathercock/internal/workers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCmdCreateEmbeddingParsing(t *testing.T) {
	tcs := []struct {
		Name      string                      `json:"-"`
		TestError func(t *testing.T, e error) `json:"-"`
		TaskID    uuid.UUID                   `json:"task_id"`
		UserID    any                         `json:"user_id"`
		EventAt   int64                       `json:"event_at"`
		EmbedType any                         `json:"embed_type"`
	}{
		{
			Name:      "Valid Query Embed Cmd",
			TestError: nil,
			TaskID:    uuid.New(),
			UserID:    1,
			EventAt:   time.Now().Unix(),
			EmbedType: "query",
		},
		{
			Name:      "Valid Passage Embed Cmd",
			TestError: nil,
			TaskID:    uuid.New(),
			UserID:    2,
			EventAt:   time.Now().Unix(),
			EmbedType: "passage",
		},
		{
			Name: "Invalid Embed Cmd",
			TestError: func(t *testing.T, e error) {
				require.ErrorIs(t, e, workers.ErrInvalidEmbedType)
			},
			TaskID:    uuid.New(),
			UserID:    3,
			EventAt:   time.Now().Unix(),
			EmbedType: "invalid",
		},
		{
			Name: "Invalid UserID type",
			TestError: func(t *testing.T, e error) {
				require.Contains(t, e.Error(), "cannot unmarshal string into Go struct field")
			},
			TaskID:    uuid.New(),
			UserID:    "invalid",
			EventAt:   time.Now().Unix(),
			EmbedType: "query",
		},
		{
			Name: "Invalid EmbedType type",
			TestError: func(t *testing.T, e error) {
				require.Contains(t, e.Error(), "cannot unmarshal number into Go struct field")
			},
			TaskID:    uuid.New(),
			UserID:    1,
			EventAt:   time.Now().Unix(),
			EmbedType: 123,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			data, err := json.Marshal(tc)
			require.NoError(t, err, "Failed to marshal test case")

			var msg workers.CmdCreateEmbedding
			err = json.Unmarshal(data, &msg)
			if tc.TestError != nil {
				tc.TestError(t, err)
				return
			}
			require.NoError(t, err, "Expected no error for valid embed type")
			serialize, err := json.Marshal(msg)
			require.NoError(t, err, "Failed to marshal CmdCreateEmbedding")
			require.Equal(t, serialize, data)
		})
	}
}

func TestCmdTasklogParsing(t *testing.T) {
	tcs := []struct {
		Name      string                      `json:"-"`
		TestError func(t *testing.T, e error) `json:"-"`
		TaskID    uuid.UUID                   `json:"task_id"`
		UserID    any                         `json:"user_id"`
		EventAt   int64                       `json:"event_at"`
		Level     any                         `json:"level"`
		Message   any                         `json:"message"`
	}{
		{
			Name:      "Valid Log Cmd",
			TestError: nil,
			TaskID:    uuid.New(),
			UserID:    1,
			EventAt:   time.Now().Unix(),
			Level:     workers.InfoLogLevel,
			Message:   "This is a log message",
		},
		{
			Name: "Invalid Log Level",
			TestError: func(t *testing.T, e error) {
				require.ErrorIs(t, e, workers.ErrInvalidLogLevel)
			},
			TaskID:  uuid.New(),
			UserID:  2,
			EventAt: time.Now().Unix(),
			Level:   "Invalid",
			Message: "This is a log message",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			data, err := json.Marshal(tc)
			require.NoError(t, err, "Failed to marshal test case")

			var msg workers.CmdTaskLog
			err = json.Unmarshal(data, &msg)
			if tc.TestError != nil {
				tc.TestError(t, err)
				return
			}
			require.NoError(t, err, "Expected no error for valid log level")
			serialize, err := json.Marshal(msg)
			require.NoError(t, err, "Failed to marshal CmdTaskLog")
			require.Equal(t, serialize, data)
		})
	}
}
