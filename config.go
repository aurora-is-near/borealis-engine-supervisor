package main

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	PromotheusURL         string
	MetricName            string
	MetricDelta           int64
	WarmupDurationSeconds int64
	CheckDurationSeconds  int64
	FailSignal            int
	HangSignal            int
}

const envPrefix = "SUPERVISOR"

/*
SUPERVISOR_PROMURL: Address of the prometheus metrics exporter (http://127.0.0.1:8041)
SUPERVISOR_METRIC: Name of the metric to test.
SUPERVISOR_WARMUPDURATION: Seconds of warmup period.
SUPERVISOR_CHECKDURATION: Seconds between metric checks.
SUPERVISOR_METRICDELTA: Expected metric delta between checks.
SUPERVISOR_FAILSIGNAL: Signal to send if no increment of metric can be detected between first and second check.
SUPERVISOR_HANGSIGNAL: Signal to send if no increment of metric can be detected between i and i+1 check (i > 1).
*/
var envVars = [...]string{"PROMURL", "METRIC", "METRICDELTA", "WARMUPDURATION", "CHECKDURATION", "FAILSIGNAL", "HANGSIGNAL"}

func ReadConfigFromEnv() (*Config, error) {
	conf := viper.New()
	conf.SetEnvPrefix(envPrefix)
	conf.AllowEmptyEnv(false)
	for _, name := range envVars {
		conf.MustBindEnv(name)
		if !conf.IsSet(name) {
			return nil, fmt.Errorf(`Environment variable "%s_%s" is not set`, envPrefix, name)
		}
	}
	return &Config{
		PromotheusURL:         conf.GetString("PROMURL"),
		MetricDelta:           conf.GetInt64("METRICDELTA"),
		MetricName:            conf.GetString("METRIC"),
		WarmupDurationSeconds: conf.GetInt64("WARMUPDURATION"),
		CheckDurationSeconds:  conf.GetInt64("CHECKDURATION"),
		FailSignal:            conf.GetInt("FAILSIGNAL"),
		HangSignal:            conf.GetInt("HANGSIGNAL"),
	}, nil
}
