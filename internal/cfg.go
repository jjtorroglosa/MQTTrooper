package internal

import (
	"flag"
	"log"
)

func GetCfg() Config {
	var dryRun = flag.Bool("d", false, "Don't run the commands. For testing purposes")
	var configFile = flag.String("c", "config.yaml", "The path to the config.yaml file")

	var mqttUser = flag.String("user", "", "MQTT user")
	var mqttPassword = flag.String("password", "", "MQTT Password")

	var httpPort = flag.Int("p", -1, "Port to listen for HTTP requests")
	var httpBindAddress = flag.String("b", "", "Address to bind HTTP server to")
	var httpAllowedAddress = flag.String("allow", "", "Address to allow HTTP requests from")

	flag.Parse()

	if *dryRun {
		log.Println("** Dry run mode **")
	}

	cfg := LoadConfigFile(*configFile)
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
	validateCfg(cfg)
	return cfg
}

func validateCfg(cfg Config) {
	if cfg.Mqtt.Enabled {
		if cfg.Mqtt.Address == "" ||
			cfg.Mqtt.Topic == "" ||
			cfg.Mqtt.User == "" ||
			cfg.Mqtt.Pass == "" ||
			cfg.Mqtt.ClientID == "" {
			panic("Invalid cfg, some mqtt fields are missing")
		}
	}
}
