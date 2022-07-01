package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestWatchMetricsSendsFailSignal(t *testing.T) {
	tickC := make(chan struct{})
	tickSleep := func() { <-tickC }
	warmUpSleep := func() { <-tickC }
	requiredDelta := int64(1)
	fail := make(chan struct{})
	sendFailSignal := func() error {
		fail <- struct{}{}
		return nil
	}
	sendHangSignal := func() error {
		t.Fatal("hang signal should not be sent")
		return nil
	}
	getMetrics := func() (int64, error) {
		return 0, nil
	}

	go watchMetrics(tickSleep, warmUpSleep, requiredDelta, sendFailSignal, sendHangSignal, getMetrics)
	tickC <- struct{}{}

	select {
	case <-fail:
	case <-time.After(5 * time.Millisecond):
		t.Fatalf("fail signal has not been called")
	}
}

func TestWatchMetricsSendsHangThenFailSignal(t *testing.T) {
	tickC := make(chan struct{}, 1)
	warmupC := make(chan struct{}, 1)
	tickSleep := func() { <-tickC }
	warmUpSleep := func() { <-warmupC }
	requiredDelta := int64(1)
	fail := make(chan struct{})
	sendFailSignal := func() error {
		fail <- struct{}{}
		return nil
	}
	hang := make(chan struct{})
	sendHangSignal := func() error {
		hang <- struct{}{}
		return nil
	}
	go watchMetrics(tickSleep, warmUpSleep, requiredDelta, sendFailSignal, sendHangSignal, createGetMetrics([]int64{0, 1, 1, 1}))
	// advance initial warmupsleep, progress has been made
	warmupC <- struct{}{}
	// advance after one tick, no progress in metrics
	tickC <- struct{}{}
	// hang signal should be sent since we have previous progress but no more
	select {
	case <-hang:
	case <-fail:
		t.Fatal("hang signal should be sent before fail signal")
	case <-time.After(5 * time.Millisecond):
		t.Fatalf("hang signal has not been sent")
	}
	// advance warmupsleep after hang signal has been sent
	warmupC <- struct{}{}
	// still no progress in metrics, a fail signal should have been sent
	select {
	case <-fail:
	case <-hang:
		t.Fatal("fail signal should have been sent after hang signal")
	case <-time.After(5 * time.Millisecond):
		t.Fatalf("fail signal has not been sent")
	}
}

func TestWatchMetricsSendsHangThenRecovers(t *testing.T) {
	tickC := make(chan struct{}, 1)
	warmupC := make(chan struct{}, 1)
	tickSleep := func() { <-tickC }
	warmUpSleep := func() { <-warmupC }
	requiredDelta := int64(1)
	fail := make(chan struct{})
	sendFailSignal := func() error {
		fail <- struct{}{}
		return nil
	}
	hang := make(chan struct{})
	sendHangSignal := func() error {
		hang <- struct{}{}
		return nil
	}
	go watchMetrics(tickSleep, warmUpSleep, requiredDelta, sendFailSignal, sendHangSignal, createGetMetrics([]int64{0, 1, 1, 2, 2}))
	// advance initial warmupsleep, progress has been made
	warmupC <- struct{}{}
	// advance after one tick, no progress in metrics
	tickC <- struct{}{}
	// hang signal should be sent since we have previous progress but no more
	select {
	case <-hang:
	case <-fail:
		t.Fatal("hang signal should be sent")
	case <-time.After(5 * time.Millisecond):
		t.Fatalf("hang signal has not been sent")
	}
	// advance warmupsleep after hang signal has been sent
	warmupC <- struct{}{}
	// new progress in metrics, no signal should be sent
	select {
	case <-fail:
		t.Fatal("fail signal should not be sent")
	case <-hang:
		t.Fatal("hang signal should not be sent")
	case <-time.After(5 * time.Millisecond):
		// nothing should be sent
	}
	//advance again after new progress
	tickC <- struct{}{}
	// metrics are again stuck, another hang signal should be sent
	select {
	case <-hang:
	case <-fail:
		t.Fatal("hang signal should be sent")
	case <-time.After(5 * time.Millisecond):
		t.Fatalf("hang signal has not been sent")
	}
}

func createGetMetrics(vals []int64) func() (int64, error) {
	idx := 0
	// create closure over idx so that returned val advances between calls
	return func() (int64, error) {
		defer func() { idx++ }()
		if idx > len(vals)-1 {
			return 0, errors.New("no more metric values")
		}
		return vals[idx], nil
	}
}

func TestWaitCommandReturnsExitCode(t *testing.T) {
	ttable := []struct {
		wantCode int
		cmd      *exec.Cmd
	}{
		{
			wantCode: 0,
			cmd:      exec.Command("sh", "-c", `exit 0`),
		},
		{
			wantCode: 15,
			cmd:      exec.Command("sh", "-c", `exit 15`),
		},
		{
			wantCode: 255,
			cmd:      exec.Command("sh", "-c", `exit 255`),
		},
	}
	for _, tc := range ttable {
		t.Run(fmt.Sprintf("exit code %v", tc.wantCode), func(t *testing.T) {
			if err := tc.cmd.Start(); err != nil {
				t.Fatalf("Failed to start subprocess: %v", err)
			}
			ch := make(chan os.Signal)
			res := waitCommand(tc.cmd, ch)
			if res != tc.wantCode {
				t.Fatalf("waitCommand did not return exit code of process, got: %v, want: %v", res, tc.wantCode)
			}
		})
	}
}

func TestWaitCommandExitsOnSignal(t *testing.T) {
	cmd := exec.Command("sh", "-c", `sleep 1`)
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start subprocess: %v", err)
	}
	sig := make(chan os.Signal)
	out := make(chan int)
	go func() {
		out <- waitCommand(cmd, sig)
	}()
	select {
	case <-out:
		t.Fatal("waitCommand should not have returned a value yet")
	case <-time.After(5 * time.Millisecond):
	}

	sig <- syscall.SIGABRT
	select {
	case <-out:
		t.Fatal("waitCommand should not exited after SIGABRT")
	case <-time.After(5 * time.Millisecond):
	}

	sig <- syscall.SIGQUIT
	select {
	case res := <-out:
		if res != 0 {
			t.Fatalf("waitCommand should have returned exit code: %v, got: %v", 0, res)
		}
	case <-time.After(5 * time.Millisecond):
		t.Fatal("waitCommand should have exited after SIGQUIT")
	}
}
