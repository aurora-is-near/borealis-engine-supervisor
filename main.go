package main

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	retcode := 0
	defer func() { os.Exit(retcode) }()

	out := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339, NoColor: true}
	log.Logger = log.Output(out)

	config, err := ReadConfigFromEnv()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGINT)

	promMetrics := NewPromotheusClient(config.PromotheusURL, config.MetricName)

	args := os.Args[1:]
	if len(args) < 1 {
		log.Fatal().Msg("Too few CLI args, need at least 1 to start the subprocess")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ() // pass current environment to the subprocess

	if err := cmd.Start(); err != nil {
		log.Fatal().Msgf("Failed to start subprocess: %v", err)
	}

	log.Info().Msgf("Started subprocess with PID: %d", cmd.Process.Pid)

	sendFailSignal := func() error {
		return cmd.Process.Signal(syscall.Signal(config.FailSignal))
	}

	sendHangSignal := func() error {
		return cmd.Process.Signal(syscall.Signal(config.HangSignal))
	}

	tickSleep := func() {
		<-time.After(time.Duration(config.CheckDurationSeconds) * time.Second)
	}

	warmUpSleep := func() {
		<-time.After(time.Duration(config.WarmupDurationSeconds) * time.Second)
	}

	go watchMetrics(tickSleep, warmUpSleep, config.MetricDelta, sendFailSignal, sendHangSignal, promMetrics.Get)
	retcode = waitCommand(cmd, interrupt)
}

func waitCommand(cmd *exec.Cmd, interrupt <-chan os.Signal) int {
	c := make(chan int)
	go func() {
		if err := cmd.Wait(); err == nil {
			c <- 0
		} else if exiterr, ok := err.(*exec.ExitError); !ok {
			c <- 0
		} else if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			c <- status.ExitStatus()
		}
	}()
	for {
		select {
		case v := <-interrupt:
			if v == syscall.SIGQUIT || v == syscall.SIGTERM {
				if err := cmd.Process.Kill(); err != nil {
					log.Error().Msgf("Failed to kill subprocess: %v", err)
				}
				return 0
			}
		case x := <-c:
			return x
		}
	}
}

/*
start -> wait_warmup -> progress -> progress -> progress -> stuck -> send_hang_signal -> wait_warmup -> progress -> progress -> ...
start -> wait_warmup -> progress -> progress -> stuck -> send_hang_signal -> wait_warmup -> stuck -> send_fail_signal
start -> wait_warmup -> stuck -> send_fail_signal
*/
func watchMetrics(tickSleep, warmUpSleep func(), requiredDelta int64, sendFailSignal, sendHangSignal func() error, getMetrics func() (int64, error)) {
	var prevProgress bool
	prev, err := getMetrics()
	if err != nil {
		log.Error().Msgf("Failed to fetch metrics data: %v", err)
	}
	warmUpSleep()
	for {
		curr, err := getMetrics()
		if err != nil {
			log.Error().Msgf("Failed to fetch metrics data: %v", err)
			tickSleep()
			continue
		}
		delta := curr - prev
		hasProgressed := delta >= requiredDelta
		if !hasProgressed && !prevProgress {
			if err := sendFailSignal(); err != nil {
				log.Error().Msgf("Failed to send fail signal: %v", err)
			}
			warmUpSleep()
		} else if !hasProgressed {
			if err := sendHangSignal(); err != nil {
				log.Error().Msgf("Failed to send hang signal: %v", err)
			}
			warmUpSleep()
		} else {
			tickSleep()
		}
		prev = curr
		prevProgress = hasProgressed
	}
}
