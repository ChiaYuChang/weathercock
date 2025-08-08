package global_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ChiaYuChang/weathercock/internal/global"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
)

func TestReadNATSConfig(t *testing.T) {
	// Create a temporary .env file for testing
	tmp := "._tmp"
	f, err := os.Create(tmp)
	require.NoError(t, err, "Failed to create temporary .env file")
	defer f.Close()

	host := "host.for.test"
	port := "4223"
	user := "test"
	pass := "password"
	_, err = f.WriteString(fmt.Sprintf("NATS_HOST=%s\n", host))
	require.NoError(t, err, "Failed to write NATS_HOST to temporary .env file")
	_, err = f.WriteString(fmt.Sprintf("NATS_CLI_PORT=%s\n", port))
	require.NoError(t, err, "Failed to write NATS_CLI_PORT to temporary .env file")
	_, err = f.WriteString(fmt.Sprintf("NATS_USER=%s\n", user))
	require.NoError(t, err, "Failed to write NATS_USER to temporary .env file")
	_, err = f.WriteString(fmt.Sprintf("NATS_PASS=%s\n", pass))
	require.NoError(t, err, "Failed to write NATS_PASS to temporary .env file")

	// Ensure the file is closed and removed after the test
	defer func() {
		err := os.Remove(tmp)
		if err != nil {
			t.Logf("Warning: Failed to remove temporary .env file: %v", err)
			t.Logf("Please clean up %s file manually", tmp)
		}
	}()

	err = global.ReadDotEnvFile(tmp, "env", []string{"."})
	require.NoError(t, err, "Failed to read .env file")

	config := global.LoadNATSConfig()
	err = config.Validate()
	require.NoError(t, err, "NATS config validation failed")

	data, err := config.MarshalJSON()
	require.NoError(t, err, "Failed to marshal NATS config to JSON")
	require.True(t, bytes.Contains(data, []byte(fmt.Sprintf(`"password":"%s"`, strings.Repeat("*", len(pass))))))
	require.Equal(t, host, config.Host, "NATS host should match")
	require.Equal(t, port, fmt.Sprintf("%d", config.Port), "NATS port should match")
	require.Equal(t, user, config.Username, "NATS user should match")
	require.Equal(t, pass, config.Password, "NATS password should match")

	var c *nats.Conn
	config.Username = ""
	require.Error(t, config.Validate(), "NATS config validation should fail when username is empty")
	config.Username = user

	config.Password = ""
	require.Error(t, config.Validate(), "NATS config validation should fail when password is empty")
	c, err = config.Connect()
	require.Error(t, err, "Should fail to connect to NATS server with invalid config")
	require.Contains(t, err.Error(), "required_without")
	require.Nil(t, c, "NATS connection should be nil")

	config.Token = "token"
	require.NoError(t, config.Validate(), "NATS config validation should pass when token is set")

	c, err = config.Connect()
	require.Error(t, err, "Should fail to connect to NATS server with invalid config")
	require.Contains(t, err.Error(), "no such host")
	require.Nil(t, c, "NATS connection should be nil")

	config.Host = ""
	require.Error(t, config.Validate(), "NATS config validation should fail when host is empty")
}

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

func TestLoadConfigsAndSingletons(t *testing.T) {
	// Default mode should be "dev"
	require.Equal(t, global.Mode(), "dev")

	err := global.LoadConfigs("NotExistFile", "env", []string{"../../"})
	require.Error(t, err, "LoadConfigs should return an error for non-existent file")

	global.Cache.Store("test_key", "test_value")
	require.Panics(t, func() { global.NATS() })
	require.Panics(t, func() { global.PostgresPool() })
	require.Panics(t, func() { global.Templates() })

	// Use a test .env file or mock config as needed
	global.Reset() // reset everything before loading new configs
	err = global.LoadConfigs(".env", "env", []string{"../../"})
	require.NoError(t, err, "LoadConfigs should not return an error")
	require.NotNil(t, global.Validator(), "Validator should not be nil")
	require.NotNil(t, global.Config(), "Config should not be nil")
	require.NotNil(t, global.NATS(), "NATS should not be nil")
	require.NotNil(t, global.PostgresPool(), "PostgresPool should not be nil")
	require.NotNil(t, global.Templates(), "Templates should not be nil")

	global.CleanUp()

	require.Nil(t, global.Validator(), "Validator should be nil after cleanup")
	require.Nil(t, global.Config(), "Config should be nil after cleanup")
	require.Nil(t, global.NATS(), "NATS should be nil after cleanup")
	require.Nil(t, global.PostgresPool(), "PostgresPool should be nil after cleanup")
	require.Nil(t, global.Templates(), "Templates should be nil after cleanup")
}
