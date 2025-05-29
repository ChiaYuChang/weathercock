package global_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/stretchr/testify/require"
)

func TestReadPostgresConfig(t *testing.T) {
	// Create a temporary password file for testing
	tmp := "._tmp"
	pwd := "39003c8c2224e265ee57a7b5f0ea47bd69"
	f, err := os.Create(tmp)
	require.NoError(t, err, "Failed to create temporary password file")
	defer f.Close()

	_, err = f.WriteString(pwd)
	require.NoError(t, err, "Failed to write to temporary password file")
	// Ensure the file is closed and removed after the test
	defer func() {
		err := os.Remove(tmp)
		if err != nil {
			t.Logf("Warning: Failed to remove temporary password file: %v", err)
		}
	}()

	// Test cases for validating PostgresConfig.
	// Name: A descriptive name for the test case.
	// PassValidate: Indicates whether the configuration is expected to pass validation.
	// PostgresConfig: The configuration to be tested.
	tcs := []struct {
		Name         string
		PassValidate bool
		global.PostgresConfig
	}{
		{
			"Use password file",
			true,
			global.PostgresConfig{
				Host:         "localhost",
				Port:         5432,
				Username:     "postgres",
				Password:     "",
				PasswordFile: tmp,
				Database:     "app",
				SSLMode:      false,
			},
		},
		{
			"Use password",
			true,
			global.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: pwd,
				Database: "app",
				SSLMode:  false,
			},
		},
		{
			"Missing password file and password",
			false,
			global.PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Database: "app",
				SSLMode:  false,
			},
		},
		{
			"Missing Host",
			false,
			global.PostgresConfig{
				Port:     5432,
				Username: "postgres",
				Password: pwd,
				Database: "app",
				SSLMode:  false,
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			data, err := tc.MarshalJSONPlain()
			require.NoError(t, err, "Failed to marshal postgres config")

			var config global.PostgresConfig
			err = json.Unmarshal(data, &config)
			require.NoError(t, err, "Failed to unmarshal postgres config")

			err = config.Validate()
			if !tc.PassValidate {
				require.Error(t, err, "Postgres config validation should fail")
				return
			}
			require.NoError(t, err, "Postgres config validation failed")

			err = config.ReadPasswordFile()
			require.NoError(t, err, "Failed to read password file")

			require.Equal(t, tc.Host, config.Host, "Host should match")
			require.Equal(t, tc.Port, config.Port, "Port should match")
			require.Equal(t, tc.Username, config.Username, "Username should match")
			require.Equal(t, tc.Database, config.Database, "Database should match")
			require.Equal(t, pwd, config.Password, "Password should match")
		})
	}
}
