package internal

import (
	"encoding/base64"
	"flag"
)

func GetCfg() (*Config, error) {
	var dryRun = flag.Bool("d", false, "Don't run the commands. For testing purposes")
	var configFile = flag.String("c", "config.yaml", "The path to the config.yaml file")

	var mqttUser = flag.String("user", "", "MQTT user")
	var mqttPassword = flag.String("password", "", "MQTT Password")

	var httpPort = flag.Int("p", -1, "Port to listen for HTTP requests")
	var httpBindAddress = flag.String("b", "", "Address to bind HTTP server to")
	var httpAllowedAddress = flag.String("allow", "", "Address to allow HTTP requests from")

	flag.Parse()

	cfg, err := LoadConfigFile(*configFile)
	if err != nil {
		return nil, err
	}
	cfg.Executor.DryRun = *dryRun
	if *mqttUser != "" {
		cfg.Mqtt.User = *mqttUser
	}
	if *mqttPassword != "" {
		cfg.Mqtt.Pass = *mqttPassword
	}

	if *httpPort != -1 {
		cfg.Http.Port = *httpPort
	}
	if *httpBindAddress != "" {
		cfg.Http.BindAddress = *httpBindAddress
	}

	if *httpAllowedAddress != "" {
		cfg.Http.AllowedAddress = *httpAllowedAddress
	}
	validateMqttConfig(cfg.Mqtt)
	csrfKey, err := base64.StdEncoding.DecodeString(cfg.Http.CsrfSecretBase64)
	cfg.Http.CsrfSecret = csrfKey

	return cfg, nil
}

func validateMqttConfig(cfg MqttConfig) {
	if cfg.Enabled {
		if cfg.Address == "" ||
			cfg.Topic == "" ||
			cfg.User == "" ||
			cfg.Pass == "" ||
			cfg.ClientID == "" {
			panic("Invalid cfg, some mqtt fields are missing")
		}
	}
}
