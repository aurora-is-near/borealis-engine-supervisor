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

	config := ReadConfigFromEnv()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGINT)

	promMetrics := NewPrometheusClient(config.PrometheusURL, config.MetricName)
	metricsGet := func() (int64, error) {
		i, err := promMetrics.Get()
		if err != nil {
			log.Err(err).Msgf("Failed to get metrics: %s", err)
			return 0, err
		}
		return i, nil
	}

	args := os.Args[1:]
	if len(args) < 1 {
		log.Fatal().Msg("Too few CLI args, need at least 1 to start the subprocess")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ() // pass current environment to the subprocess
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

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

	go watchMetrics(tickSleep, warmUpSleep, config.MetricDelta, sendFailSignal, sendHangSignal, metricsGet)
	retcode = waitCommand(cmd, interrupt)
}

// waitCommand waits until a command is finished and returns its exit code.
// Can be interrupted by sending a SIGQUIT or SIGTERM over the interrupt channel,
// the subprocess is killed and exit code 0 is returned.
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
			if v != syscall.SIGQUIT && v != syscall.SIGTERM {
				continue
			}
			if err := cmd.Process.Kill(); err != nil {
				log.Error().Msgf("Failed to kill subprocess: %v", err)
			}
			return 0
		case x := <-c:
			return x
		}
	}
}

// watchMetrics fetches metric data and ensures that it has progress by at least requiredDelta.
// If no progress is made, a hang signal is sent and if progress doesn't pick back up, a fail signal is sent.
//
// start -> wait_warmup -> progress -> progress -> progress -> stuck -> send_hang_signal -> wait_warmup -> progress -> progress -> ...
// start -> wait_warmup -> progress -> progress -> stuck -> send_hang_signal -> wait_warmup -> stuck -> send_fail_signal
// start -> wait_warmup -> stuck -> send_fail_signal
func watchMetrics(tickSleep, warmUpSleep func(), requiredDelta int64, sendFailSignal, sendHangSignal func() error, getMetrics func() (int64, error)) {
	var prevProgress bool
	prev, _ := getMetrics()
	warmUpSleep()
	for {
		curr, _ := getMetrics()
		delta := curr - prev
		hasProgressed := delta >= requiredDelta
		if !hasProgressed && !prevProgress {
			log.Info().Msg("Supervisor: Engine not connected")
			if err := sendFailSignal(); err != nil {
				log.Error().Msgf("Failed to send fail signal: %v", err)
			}
			warmUpSleep()
		} else if !hasProgressed {
			log.Info().Msg("Supervisor: Engine falling behind")
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
