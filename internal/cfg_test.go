package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCfgMqttEnabled(t *testing.T) {
	baseCfg := Config{
		Mqtt: MqttConfig{
			Enabled:  true,
			Address:  "tcp://localhost:1883",
			Topic:    "test",
			User:     "user",
			Pass:     "pass",
			ClientID: "client",
		},
	}

	t.Run("should not panic when all mqtt fields are present", func(t *testing.T) {
		assert.NotPanics(t, func() {
			validateMqttConfig(baseCfg.Mqtt)
		})
	})

	t.Run("should panic when address is missing", func(t *testing.T) {
		cfg := baseCfg
		cfg.Mqtt.Address = ""
		assert.PanicsWithValue(t, "Invalid cfg, some mqtt fields are missing", func() {
			validateMqttConfig(cfg.Mqtt)
		})
	})

	t.Run("should panic when topic is missing", func(t *testing.T) {
		cfg := baseCfg
		cfg.Mqtt.Topic = ""
		assert.PanicsWithValue(t, "Invalid cfg, some mqtt fields are missing", func() {
			validateMqttConfig(cfg.Mqtt)
		})
	})

	t.Run("should panic when user is missing", func(t *testing.T) {
		cfg := baseCfg
		cfg.Mqtt.User = ""
		assert.PanicsWithValue(t, "Invalid cfg, some mqtt fields are missing", func() {
			validateMqttConfig(cfg.Mqtt)
		})
	})

	t.Run("should panic when pass is missing", func(t *testing.T) {
		cfg := baseCfg
		cfg.Mqtt.Pass = ""
		assert.PanicsWithValue(t, "Invalid cfg, some mqtt fields are missing", func() {
			validateMqttConfig(cfg.Mqtt)
		})
	})

	t.Run("should panic when clientID is missing", func(t *testing.T) {
		cfg := baseCfg
		cfg.Mqtt.ClientID = ""
		assert.PanicsWithValue(t, "Invalid cfg, some mqtt fields are missing", func() {
			validateMqttConfig(cfg.Mqtt)
		})
	})
}

func TestValidateCfgMqttDisabled(t *testing.T) {
	baseCfg := Config{
		Mqtt: MqttConfig{
			Enabled: false,
		},
	}

	t.Run("should not panic when mqtt is disabled", func(t *testing.T) {
		assert.NotPanics(t, func() {
			validateMqttConfig(baseCfg.Mqtt)
		})
	})
}
