package internal

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfigFile(t *testing.T) {
	// Create a temporary YAML file for testing
	content := `
http:
  enabled: true
  port: 8080
  allowed_address: "127.0.0.1"
  bind_address: "0.0.0.0"
  csrf_secret: "***REMOVED***="
mqtt:
  enabled: true
  address: "tcp://127.0.0.1:1883"
  client_id: "mqttrooper_client_id"
  connection_timeout_seconds: 4
  user: "user"
  pass: "pass"
  topic: "/test"
executor:
  shell: "/bin/bash"
  dry_run: false
services:
  service1: "command1"
  service2: "command2"

daemon:
  cwd: "any/cwd"
  env_path: "any:path_variable"
  log_file_path: "info_log_file"
  error_file_path: "error_log_file"
  mac_id: "com.some.id"
`
	tmpfile, err := os.CreateTemp("", "test.yaml")
	assert.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()

	_, err = tmpfile.WriteString(content)
	assert.NoError(t, err)
	err = tmpfile.Close()
	assert.NoError(t, err)

	// Call LoadConfigFile with the temporary file
	cfg, err := LoadConfigFile(tmpfile.Name())
	assert.NoError(t, err)

	// Assert that the returned Config struct contains the expected values
	assert.Equal(t, true, cfg.Http.Enabled)
	assert.Equal(t, 8080, cfg.Http.Port)
	assert.Equal(t, "127.0.0.1", cfg.Http.AllowedAddress)
	assert.Equal(t, "0.0.0.0", cfg.Http.BindAddress)
	assert.Equal(t, "hello world", string(cfg.Http.CsrfSecret))

	assert.Equal(t, true, cfg.Mqtt.Enabled)
	assert.Equal(t, "tcp://127.0.0.1:1883", cfg.Mqtt.Address)
	assert.Equal(t, "mqttrooper_client_id", cfg.Mqtt.ClientID)
	assert.Equal(t, "user", cfg.Mqtt.User)
	assert.Equal(t, "pass", cfg.Mqtt.Pass)
	assert.Equal(t, "/test", cfg.Mqtt.Topic)

	assert.Equal(t, "/bin/bash", cfg.Executor.Shell)
	assert.Equal(t, false, cfg.Executor.DryRun)

	assert.Len(t, cfg.Services, 2)
	assert.Equal(t, "command1", cfg.Services["service1"])
	assert.Equal(t, 4, cfg.Mqtt.ConnectionTimeoutSeconds)
	assert.Equal(t, "command2", cfg.Services["service2"])

	assert.Len(t, cfg.ServicesList, 2)
	assert.Equal(t, "service1", cfg.ServicesList[0].Name)
	assert.Equal(t, "command1", cfg.ServicesList[0].Command)
	assert.Equal(t, "service2", cfg.ServicesList[1].Name)
	assert.Equal(t, "command2", cfg.ServicesList[1].Command)

	assert.Equal(t, "any/cwd", cfg.Daemon.Cwd)
	assert.Equal(t, "any:path_variable", cfg.Daemon.EnvPath)

	assert.Equal(t, "info_log_file", cfg.Daemon.LogFilePath)
	assert.Equal(t, "error_log_file", cfg.Daemon.ErrorFilePath)
	assert.Equal(t, "com.some.id", cfg.Daemon.MacID)
}

func TestLoadConfigFile_FileNotFound(t *testing.T) {
	// Call openFile with a non-existent file
	_, err := openFile("non-existent-file.yaml")

	// Assert that an error is returned
	assert.Error(t, err)
}

func TestLoadConfigFileHttpDefaults(t *testing.T) {
	// Create a temporary YAML file for testing
	content := `
http:
  enabled: true
`
	tmpfile, err := os.CreateTemp("", "test.yaml")
	assert.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()

	_, err = tmpfile.WriteString(content)
	assert.NoError(t, err)
	err = tmpfile.Close()
	assert.NoError(t, err)

	// Call LoadConfigFile with the temporary file
	cfg, err := LoadConfigFile(tmpfile.Name())
	assert.NoError(t, err)

	// Assert that the returned Config struct contains the expected values
	assert.Equal(t, true, cfg.Http.Enabled)
	assert.Equal(t, 8080, cfg.Http.Port)
	assert.Equal(t, "127.0.0.1", cfg.Http.AllowedAddress)
	assert.Equal(t, "127.0.0.1", cfg.Http.BindAddress)
}
