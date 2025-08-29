package global_test

import (
	"strings"
	"testing"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	type TestConfig struct {
		Name string `json:"name"`
		Port int    `json:"port"`
	}

	tcs := []struct {
		name         string
		configJSON   string
		configType   string
		expected     TestConfig
		expectErr    bool
		expectedMode string
	}{
		{
			name:         "Valid JSON config",
			configJSON:   `{"name": "test-app", "port": 8080, "mode": "prod"}`,
			configType:   "json",
			expected:     TestConfig{Name: "test-app", Port: 8080},
			expectErr:    false,
			expectedMode: "prod",
		},
		{
			name:         "Valid YAML config",
			configJSON:   `name: test-app\nport: 8080\nmode: dev`,
			configType:   "yaml",
			expected:     TestConfig{Name: "test-app", Port: 8080},
			expectErr:    false,
			expectedMode: "dev",
		},
		{
			name:         "Default mode",
			configJSON:   `{"name": "test-app", "port": 8080}`,
			configType:   "json",
			expected:     TestConfig{Name: "test-app", Port: 8080},
			expectErr:    false,
			expectedMode: "dev",
		},
		{
			name:       "Invalid JSON",
			configJSON: `{"name": "test-app", "port": 8080,}`,
			configType: "json",
			expectErr:  true,
		},
		{
			name:       "Mismatched type",
			configJSON: `{"name": "test-app", "port": "8080"}`,
			configType: "json",
			expectErr:  true,
		},
		{
			name:       "Nil reader",
			configJSON: "",
			configType: "json",
			expectErr:  true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var cfg TestConfig
			var err error

			if tc.name == "Nil reader" {
				err = global.LoadConfig(nil, tc.configType, &cfg)
			} else {
				r := strings.NewReader(tc.configJSON)
				err = global.LoadConfig(r, tc.configType, &cfg)
			}

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, cfg)
				require.Equal(t, tc.expectedMode, global.Mode())
			}
		})
	}
}

func TestPostgresConfig_Validate(t *testing.T) {
	global.InitValidator()

	tcs := []struct {
		name      string
		cfg       global.PostgresConfig
		expectErr bool
	}{
		{
			name: "Valid config",
			cfg: global.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Username: "user",
				Password: "password123",
				Database: "test_db",
			},
			expectErr: false,
		},
		{
			name: "Missing host",
			cfg: global.PostgresConfig{
				Port:     5432,
				Username: "user",
				Password: "password123",
				Database: "test_db",
			},
			expectErr: true,
		},
		{
			name: "Missing password",
			cfg: global.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Username: "user",
				Database: "test_db",
			},
			expectErr: true,
		},
		{
			name: "Short password warning (should still pass)",
			cfg: global.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Username: "user",
				Password: "123",
				Database: "test_db",
			},
			expectErr: false, // Validation itself doesn't fail, it just logs a warning
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
