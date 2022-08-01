/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package config

import "os"

var Config = newEnvConfig()

type envConfig struct {
	ServerURL   string
	AccessToken string
	Workflow    string
}

func newEnvConfig() *envConfig {
	config := &envConfig{
		ServerURL:   envOrDefault("REANA_SERVER_URL", ""),
		AccessToken: envOrDefault("REANA_ACCESS_TOKEN", ""),
		Workflow:    envOrDefault("REANA_WORKON", ""),
	}
	return config
}

func envOrDefault(name, defaultValue string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return defaultValue
}
