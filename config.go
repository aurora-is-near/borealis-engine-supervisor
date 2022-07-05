package main

import (
	"github.com/spf13/viper"
)

// Config contains all the configuration options that are read from environment variables.
type Config struct {
	PrometheusURL         string
	MetricName            string
	MetricDelta           int64
	WarmupDurationSeconds int64
	CheckDurationSeconds  int64
	FailSignal            int
	HangSignal            int
}

// envPrefix is the prefix in the names of environment variables.
const envPrefix = "SUPERVISOR"

/*
defaults contains names of environment variables for supervisor configuration.
SUPERVISOR_PROMURL: Address of the prometheus metrics exporter (http://127.0.0.1:8041)
SUPERVISOR_METRIC: Name of the metric to test.
SUPERVISOR_WARMUPDURATION: Seconds of warmup period.
SUPERVISOR_CHECKDURATION: Seconds between metric checks.
SUPERVISOR_METRICDELTA: Expected metric delta between checks.
SUPERVISOR_FAILSIGNAL: Signal to send if no increment of metric can be detected between first and second check.
SUPERVISOR_HANGSIGNAL: Signal to send if no increment of metric can be detected between i and i+1 check (i > 1).
*/
var defaults = map[string]string{
	"PROMURL":        "http://127.0.0.1:8041",
	"METRIC":         "engine_last_sequential_id_received",
	"METRICDELTA":    "1",
	"WARMUPDURATION": "300",
	"CHECKDURATION":  "60",
	"FAILSIGNAL":     "1",
	"HANGSIGNAL":     "9",
}

// ReadConfigFromEnv reads all variables contained in envVars from the environment.
func ReadConfigFromEnv() *Config {
	conf := viper.New()
	conf.SetEnvPrefix(envPrefix)
	conf.AllowEmptyEnv(false)
	for k, v := range defaults {
		conf.SetDefault(k, v)
	}
	return &Config{
		PrometheusURL:         conf.GetString("PROMURL"),
		MetricDelta:           conf.GetInt64("METRICDELTA"),
		MetricName:            conf.GetString("METRIC"),
		WarmupDurationSeconds: conf.GetInt64("WARMUPDURATION"),
		CheckDurationSeconds:  conf.GetInt64("CHECKDURATION"),
		FailSignal:            conf.GetInt("FAILSIGNAL"),
		HangSignal:            conf.GetInt("HANGSIGNAL"),
	}
}
