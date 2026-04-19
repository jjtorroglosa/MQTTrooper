package internal

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func loadConfigFromContent(t *testing.T, content string) (*Config, error) {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "test.yaml")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = os.Remove(tmpfile.Name()) })
	if _, err := tmpfile.WriteString(content); err != nil {
		return nil, err
	}
	if err := tmpfile.Close(); err != nil {
		return nil, err
	}
	return LoadConfigFile(tmpfile.Name())
}

func TestLoadConfigFile(t *testing.T) {
	// Create a temporary YAML file for testing
	content := `
http:
  enabled: true
  port: 8080
  allowed_address: "127.0.0.1"
  bind_address: "0.0.0.0"
  csrf_secret: "aGVsbG8gd29ybGQ="
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

func TestLoadConfigFileDiscoveryDefaults(t *testing.T) {
	content := `
mqtt:
  enabled: true
  address: "tcp://127.0.0.1:1883"
  client_id: "c"
  user: "u"
  pass: "p"
  topic: "/t"
  discovery:
    enabled: true
`
	tmpfile, err := os.CreateTemp("", "test.yaml")
	assert.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()
	_, err = tmpfile.WriteString(content)
	assert.NoError(t, err)
	assert.NoError(t, tmpfile.Close())

	cfg, err := LoadConfigFile(tmpfile.Name())
	assert.NoError(t, err)

	assert.Equal(t, true, cfg.Mqtt.Discovery.Enabled)
	assert.Equal(t, "homeassistant", cfg.Mqtt.Discovery.Prefix)
	assert.Equal(t, "mqttrooper", cfg.Mqtt.Discovery.DevicePrefix)
	assert.Equal(t, "mqttrooper", cfg.Mqtt.Discovery.DeviceName)
}

func TestLoadConfigFileDiscoveryOverrides(t *testing.T) {
	content := `
mqtt:
  enabled: true
  discovery:
    enabled: true
    prefix: "ha"
    device_prefix: "mqttrooper_nas"
    device_name: "NAS buttons"
`
	tmpfile, err := os.CreateTemp("", "test.yaml")
	assert.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()
	_, err = tmpfile.WriteString(content)
	assert.NoError(t, err)
	assert.NoError(t, tmpfile.Close())

	cfg, err := LoadConfigFile(tmpfile.Name())
	assert.NoError(t, err)

	assert.Equal(t, "ha", cfg.Mqtt.Discovery.Prefix)
	assert.Equal(t, "mqttrooper_nas", cfg.Mqtt.Discovery.DevicePrefix)
	assert.Equal(t, "NAS buttons", cfg.Mqtt.Discovery.DeviceName)
}

func TestLoadConfigFileEntities(t *testing.T) {
	content := `
mqtt:
  enabled: true
  topic: "/t"
entities:
  snapclient:
    type: command
    run: systemctl --user restart snapclient.service
  volume:
    type: number
    min: 0
    max: 100
    step: 1
    get: pactl get-sink-volume @DEFAULT_SINK@
    set: pactl set-sink-volume @DEFAULT_SINK@ {value}%
  mute:
    type: boolean
    get: pactl get-sink-mute @DEFAULT_SINK@
    on: pactl set-sink-mute @DEFAULT_SINK@ 1
    off: pactl set-sink-mute @DEFAULT_SINK@ 0
`
	tmpfile, err := os.CreateTemp("", "test.yaml")
	assert.NoError(t, err)
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_, err = tmpfile.WriteString(content)
	assert.NoError(t, err)
	assert.NoError(t, tmpfile.Close())

	cfg, err := LoadConfigFile(tmpfile.Name())
	assert.NoError(t, err)

	assert.Len(t, cfg.Entities, 3)

	snap := cfg.Entities["snapclient"]
	assert.Equal(t, EntityTypeCommand, snap.Type)
	assert.Equal(t, "systemctl --user restart snapclient.service", snap.Run)

	vol := cfg.Entities["volume"]
	assert.Equal(t, EntityTypeNumber, vol.Type)
	assert.Equal(t, float64(0), vol.Min)
	assert.Equal(t, float64(100), vol.Max)
	assert.Equal(t, float64(1), vol.Step)
	assert.Equal(t, "pactl get-sink-volume @DEFAULT_SINK@", vol.Get)
	assert.Equal(t, "pactl set-sink-volume @DEFAULT_SINK@ {value}%", vol.Set)

	mute := cfg.Entities["mute"]
	assert.Equal(t, EntityTypeBoolean, mute.Type)
	assert.Equal(t, "pactl get-sink-mute @DEFAULT_SINK@", mute.Get)
	assert.Equal(t, "pactl set-sink-mute @DEFAULT_SINK@ 1", mute.On)
	assert.Equal(t, "pactl set-sink-mute @DEFAULT_SINK@ 0", mute.Off)
}

func TestLoadConfigFileEntitiesBackwardCompatServices(t *testing.T) {
	content := `
mqtt:
  enabled: true
  topic: "/t"
services:
  restart: echo restart
entities:
  volume:
    type: number
    min: 0
    max: 100
    step: 1
    get: echo 50
    set: echo {value}
`
	tmpfile, err := os.CreateTemp("", "test.yaml")
	assert.NoError(t, err)
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_, err = tmpfile.WriteString(content)
	assert.NoError(t, err)
	assert.NoError(t, tmpfile.Close())

	cfg, err := LoadConfigFile(tmpfile.Name())
	assert.NoError(t, err)

	// services entry folded into Entities as command type
	restart, ok := cfg.Entities["restart"]
	assert.True(t, ok)
	assert.Equal(t, EntityTypeCommand, restart.Type)
	assert.Equal(t, "echo restart", restart.Run)

	// explicit entities entry not overwritten
	assert.Equal(t, EntityTypeNumber, cfg.Entities["volume"].Type)

	// ServicesList derived from command entities + legacy services
	names := make([]string, len(cfg.ServicesList))
	for i, s := range cfg.ServicesList {
		names[i] = s.Name
	}
	assert.Contains(t, names, "restart")
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
