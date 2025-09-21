package internal

import (
	"flag"
	"fmt"
)

func GetCfg() Config {
	var dryRun = flag.Bool("d", false, "Don't run the commands. For testing purposes")
	var configFile = flag.String("c", "config.yaml", "The path to the config.yaml file")

	var mqttUser = flag.String("user", "", "Mqtt user")
	var mqttPassword = flag.String("password", "", "Mqtt password")

	var httpPort = flag.Int("p", 8080, "Port to listen for HTTP requests")
	var httpBindAddress = flag.String("b", "127.0.0.1", "Address to bind to")
	var httpAllowedAddress = flag.String("allow", "127.0.0.1", "Address to allow requests from")

	flag.Parse()

	if *dryRun {
		fmt.Println("** Dry run mode **")
	}

	cfg := LoadConfigFile(*configFile)
	cfg.Executor.DryRun = *dryRun
	if *mqttUser != "" {
		cfg.Mqtt.User = *mqttUser
	}
	if *mqttPassword != "" {
		cfg.Mqtt.Pass = *mqttPassword
	}

	cfg.Http.Port = *httpPort
	cfg.Http.BindAddress = *httpBindAddress
	cfg.Http.AllowedAddress = *httpAllowedAddress
	return cfg
}
