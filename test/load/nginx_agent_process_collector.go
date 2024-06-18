// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package load

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"text/template"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/testbed"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/process"
)

// nginxAgentProcessCollector implements the OtelcolRunner interface as a child process on the same machine executing
// the test. The process can be monitored and the output of which will be written to a slog file.
type nginxAgentProcessCollector struct {
	// Path to agent executable. If unset the default executable in
	// bin/otelcol_{{.GOOS}}_{{.GOARCH}} will be used.
	// Can be set for example to use the unstable executable for a specific test.
	agentExePath string

	// Descriptive name of the process
	name string

	// Command to execute
	cmd *exec.Cmd

	// additional env vars (os.Environ() populated by default)
	additionalEnv map[string]string

	// Various starting/stopping flags
	isStarted  bool
	stopOnce   sync.Once
	isStopped  bool
	doneSignal chan struct{}

	// Resource specification that must be monitored for.
	resourceSpec *testbed.ResourceSpec

	// Process monitoring data.
	processMon *process.Process

	// Time when process was started.
	startTime time.Time

	// Last tick time we monitored the process.
	lastElapsedTime time.Time

	// Process times that were fetched on last monitoring tick.
	lastProcessTimes *cpu.TimesStat

	// Current RAM RSS in MiBs
	ramMiBCur atomic.Uint32

	// Current CPU percentage times 1000 (we use scaling since we have to use int for atomic operations).
	cpuPercentX1000Cur atomic.Uint32

	// Maximum CPU seen
	cpuPercentMax float64

	// Number of memory measurements
	memProbeCount int

	// Cumulative RAM RSS in MiBs
	ramMiBTotal uint64

	// Maximum RAM seen
	ramMiBMax uint32
}

type NginxAgentProcessOption func(*nginxAgentProcessCollector)

const (
	mibibyte                     = 1024 * 1024
	maxCPU                       = 60
	maxRAM                       = 200
	cpuMultiplier                = 1000
	cpuDeltaNumerator            = 100
	curCPUPercentageX1000Divisor = 1000.0
	cpuPercentageX1000Divisor    = 1000
	waitPeriod                   = 10 * time.Second
	processTimeMultiplier        = 100.0
)

// NewNginxAgentProcessCollector creates a new OtelcolRunner as a child process on the same machine executing the test.
// nolint: ireturn
func NewNginxAgentProcessCollector(options ...NginxAgentProcessOption) testbed.OtelcolRunner {
	col := &nginxAgentProcessCollector{additionalEnv: make(map[string]string)}

	for _, option := range options {
		option(col)
	}

	return col
}

// WithAgentExePath sets the path of the Collector executable
func WithAgentExePath(exePath string) NginxAgentProcessOption {
	return func(cpc *nginxAgentProcessCollector) {
		cpc.agentExePath = exePath
	}
}

// WithEnvVar sets an additional environment variable for the process
func WithEnvVar(k, v string) NginxAgentProcessOption {
	return func(cpc *nginxAgentProcessCollector) {
		cpc.additionalEnv[k] = v
	}
}

func (cp *nginxAgentProcessCollector) PrepareConfig(configStr string) (configCleanup func(), err error) {
	// configCleanup = func() {
	// 	// NoOp
	// }
	// var file *os.File
	// file, err = os.CreateTemp("", "agent*.yaml")
	// if err != nil {
	// 	slog.Info("%s", err)
	// 	return configCleanup, err
	// }

	// defer func() {
	// 	errClose := file.Close()
	// 	if errClose != nil {
	// 		slog.Info("%s", errClose)
	// 	}
	// }()

	// if _, err = file.WriteString(configStr); err != nil {
	// 	slog.Info("%s", err)
	// 	return configCleanup, err
	// }
	// cp.configFileName = file.Name()
	// configCleanup = func() {
	// 	os.Remove(cp.configFileName)
	// }

	// return configCleanup, err
	return func() {}, nil
}

func expandExeFileName(exeName string) string {
	cfgTemplate, err := template.New("").Parse(exeName)
	if err != nil {
		slog.Error("Template failed to parse exe name %q: %s",
			exeName, err.Error())
	}

	templateVars := struct {
		GOOS   string
		GOARCH string
	}{
		GOOS:   runtime.GOOS,
		GOARCH: runtime.GOARCH,
	}
	var buf bytes.Buffer
	if err = cfgTemplate.Execute(&buf, templateVars); err != nil {
		slog.Error("Configuration template failed to run on exe name %q: %s",
			exeName, err.Error())
	}

	return buf.String()
}

// Start a child process.
//
// cp.AgentExePath defines the executable to run. 
//
// Parameters:
// name is the human readable name of the process (e.g. "Agent"), used for slogging.
// slogFilePath is the file path to write the standard output and standard error of
// the process to.
// cmdArgs is the command line arguments to pass to the process.
func (cp *nginxAgentProcessCollector) Start(params testbed.StartParams) error {
	cp.name = params.Name
	cp.doneSignal = make(chan struct{})
	cp.resourceSpec = &testbed.ResourceSpec{
		ExpectedMaxCPU: maxCPU,
		ExpectedMaxRAM: maxRAM,
	}

	if cp.agentExePath == "" {
		cp.agentExePath = testbed.GlobalConfig.DefaultAgentExeRelativeFile
	}
	exePath := expandExeFileName(cp.agentExePath)
	exePath, err := filepath.Abs(exePath)
	if err != nil {
		return err
	}

	slog.Info("Starting %s (%s)", cp.name, exePath)

	// Prepare slog file
	slogFile, err := os.Create(params.LogFilePath)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", params.LogFilePath, err)
	}
	slog.Info("Writing %s slog to %s", cp.name, params.LogFilePath)

	// Prepare to start the process.
	cp.cmd = exec.Command(exePath, params.CmdArgs...)
	cp.cmd.Env = os.Environ()

	// update env deterministically
	additionalEnvVars := make([]string, 0)
	for k := range cp.additionalEnv {
		additionalEnvVars = append(additionalEnvVars, k)
	}
	sort.Strings(additionalEnvVars)
	for _, k := range additionalEnvVars {
		cp.cmd.Env = append(cp.cmd.Env, fmt.Sprintf("%s=%s", k, cp.additionalEnv[k]))
	}

	// Capture standard output and standard error.
	cp.cmd.Stdout = slogFile
	cp.cmd.Stderr = slogFile

	// Start the process.
	if err = cp.cmd.Start(); err != nil {
		return fmt.Errorf("cannot start executable at %s: %w", exePath, err)
	}

	cp.startTime = time.Now()
	cp.isStarted = true

	slog.Info("%s running, pid=%d", cp.name, cp.cmd.Process.Pid)

	return err
}

func (cp *nginxAgentProcessCollector) Stop() (stopped bool, err error) {
	if !cp.isStarted || cp.isStopped {
		return false, nil
	}
	cp.stopOnce.Do(func() {
		cp.isStopped = true

		slog.Info("Gracefully terminating %s pid=%d, sending SIGTEM...", cp.name, cp.cmd.Process.Pid)

		// Notify resource monitor to stop.
		close(cp.doneSignal)

		// Gracefully signal process to stop.
		if err = cp.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			slog.Info("Cannot send SIGTEM: error", slog.String("error", err.Error()))
		}

		finished := make(chan struct{})

		// Setup a goroutine to wait a while for process to finish and send kill signal
		// to the process if it doesn't finish.
		go func() {
			t := time.After(waitPeriod)
			select {
			case <-t:
				// Time is out. Kill the process.
				slog.Info("%s pid=%d is not responding to SIGTERM. Sending SIGKILL to kill forcedly.",
					cp.name, cp.cmd.Process.Pid)
				if signalErr := cp.cmd.Process.Signal(syscall.SIGKILL); signalErr != nil {
					slog.Error("Cannot send SIGKILL:", "error", signalErr.Error())
				}
			case <-finished:
				// Process is successfully finished.
			}
		}()

		// Wait for process to terminate
		err = cp.cmd.Wait()

		// Let goroutine know process is finished.
		close(finished)

		// Set resource consumption stats to 0
		cp.ramMiBCur.Store(0)
		cp.cpuPercentX1000Cur.Store(0)

		slog.Info("%s process stopped, exit code=%d", cp.name, cp.cmd.ProcessState.ExitCode())

		if err != nil {
			slog.Info("%s execution failed: %s", cp.name, err.Error())
		}
	})

	return true, err
}

func (cp *nginxAgentProcessCollector) WatchResourceConsumption() error {
	if cp.resourceSpec != nil && (cp.resourceSpec.ExpectedMaxCPU != 0 || cp.resourceSpec.ExpectedMaxRAM != 0) {
		return nil
	}

	var err error
	cp.processMon, err = process.NewProcess(int32(cp.cmd.Process.Pid))
	if err != nil {
		return fmt.Errorf("cannot monitor process %d: %w", cp.cmd.Process.Pid, err)
	}

	cp.fetchRAMUsage()

	// Begin measuring elapsed and process CPU times.
	cp.lastElapsedTime = time.Now()
	cp.lastProcessTimes, err = cp.processMon.Times()
	if err != nil {
		return fmt.Errorf("cannot get process times for %d: %w", cp.cmd.Process.Pid, err)
	}

	// Measure every ResourceCheckPeriod.
	ticker := time.NewTicker(cp.resourceSpec.ResourceCheckPeriod)
	defer ticker.Stop()

	// on first start must be under the cpu and ram max usage add a max minute delay
	for start := time.Now(); time.Since(start) < time.Minute; {
		cp.fetchRAMUsage()
		cp.fetchCPUUsage()
		allowanceErr := cp.checkAllowedResourceUsage()
		if allowanceErr == nil {
			break
		}

		slog.Info("Allowed usage of resources is too high before test starts wait for one second : %v", allowanceErr)
		time.Sleep(time.Second)
	}

	remainingFailures := cp.resourceSpec.MaxConsecutiveFailures
	for {
		select {
		case <-ticker.C:
			cp.fetchRAMUsage()
			cp.fetchCPUUsage()

			if allowUsageErr := cp.checkAllowedResourceUsage(); allowUsageErr != nil {
				if remainingFailures > 0 {
					remainingFailures--
					slog.Info("Resource utilization too high. Remaining attempts:", "failures_left", remainingFailures)

					continue
				}
				if _, errStop := cp.Stop(); errStop != nil {
					slog.Info("Failed to stop child process: %v", errStop)
				}

				return err
			}

		case <-cp.doneSignal:
			slog.Info("Stopping process monitor.")

			return nil
		}
	}
}

func (cp *nginxAgentProcessCollector) GetProcessMon() *process.Process {
	return cp.processMon
}

func (cp *nginxAgentProcessCollector) fetchRAMUsage() {
	// Get process memory and CPU times
	mi, err := cp.processMon.MemoryInfo()
	if err != nil {
		slog.Info("cannot get process memory for pid: error",
			slog.Int("pid", cp.cmd.Process.Pid),
			slog.String("error", err.Error()))

		return
	}

	// Calculate RSS in MiBs.
	ramMiBCur := uint32(mi.RSS / mibibyte)

	// Calculate aggregates.
	cp.memProbeCount++
	cp.ramMiBTotal += uint64(ramMiBCur)
	if ramMiBCur > cp.ramMiBMax {
		cp.ramMiBMax = ramMiBCur
	}

	// Store current usage.
	cp.ramMiBCur.Store(ramMiBCur)
}

func (cp *nginxAgentProcessCollector) fetchCPUUsage() {
	times, err := cp.processMon.Times()
	if err != nil {
		slog.Info("cannot get process times for pid: error",
			slog.Int("pid", cp.cmd.Process.Pid),
			slog.String("error", err.Error()))

		return
	}

	now := time.Now()

	// Calculate elapsed and process CPU time deltas in seconds
	deltaElapsedTime := now.Sub(cp.lastElapsedTime).Seconds()
	deltaCPUTime := totalCPU(times) - totalCPU(cp.lastProcessTimes)
	if deltaCPUTime < 0 {
		// We sometimes get negative difference when the process is terminated.
		deltaCPUTime = 0
	}

	cp.lastProcessTimes = times
	cp.lastElapsedTime = now

	// Calculate CPU usage percentage in elapsed period.
	cpuPercent := deltaCPUTime * cpuDeltaNumerator / deltaElapsedTime
	if cpuPercent > cp.cpuPercentMax {
		cp.cpuPercentMax = cpuPercent
	}

	curCPUPercentageX1000 := uint32(cpuPercent * cpuMultiplier)

	// Store current usage.
	cp.cpuPercentX1000Cur.Store(curCPUPercentageX1000)
}

func (cp *nginxAgentProcessCollector) checkAllowedResourceUsage() error {
	// Check if current CPU usage exceeds expected.
	var errMsg string
	if cp.resourceSpec.ExpectedMaxCPU != 0 &&
		cp.cpuPercentX1000Cur.Load()/cpuPercentageX1000Divisor > cp.resourceSpec.ExpectedMaxCPU {
		errMsg = fmt.Sprintf("CPU consumption is %.1f%%, max expected is %d%%",
			float64(cp.cpuPercentX1000Cur.Load())/curCPUPercentageX1000Divisor, cp.resourceSpec.ExpectedMaxCPU)
	}

	// Check if current RAM usage exceeds expected.
	if cp.resourceSpec.ExpectedMaxRAM != 0 && cp.ramMiBCur.Load() > cp.resourceSpec.ExpectedMaxRAM {
		formattedCurRAM := strconv.FormatUint(uint64(cp.ramMiBCur.Load()), 10)
		errMsg = fmt.Sprintf("RAM consumption is %s MiB, max expected is %d MiB",
			formattedCurRAM, cp.resourceSpec.ExpectedMaxRAM)
	}

	if errMsg == "" {
		return nil
	}

	slog.Info("Performance error: error", slog.String("error", errMsg))

	return errors.New(errMsg)
}

// GetResourceConsumption returns resource consumption as a string
func (cp *nginxAgentProcessCollector) GetResourceConsumption() string {
	if cp.resourceSpec != nil && (cp.resourceSpec.ExpectedMaxCPU != 0 || cp.resourceSpec.ExpectedMaxRAM != 0) {
		return ""
	}

	curRSSMib := cp.ramMiBCur.Load()
	curCPUPercentageX1000 := cp.cpuPercentX1000Cur.Load()

	return fmt.Sprintf("%s RAM (RES):%4d MiB, CPU:%4.1f%%", cp.name,
		curRSSMib, float64(curCPUPercentageX1000)/curCPUPercentageX1000Divisor)
}

// GetTotalConsumption returns total resource consumption since start of process
func (cp *nginxAgentProcessCollector) GetTotalConsumption() *testbed.ResourceConsumption {
	rc := &testbed.ResourceConsumption{}

	if cp.processMon != nil {
		// Get total elapsed time since process start
		elapsedDuration := cp.lastElapsedTime.Sub(cp.startTime).Seconds()

		if elapsedDuration > 0 {
			// Calculate average CPU usage since start of process
			rc.CPUPercentAvg = totalCPU(cp.lastProcessTimes) / elapsedDuration * processTimeMultiplier
		}
		rc.CPUPercentMax = cp.cpuPercentMax

		if cp.memProbeCount > 0 {
			// Calculate average RAM usage by averaging all RAM measurements
			rc.RAMMiBAvg = uint32(cp.ramMiBTotal / uint64(cp.memProbeCount))
		}
		rc.RAMMiBMax = cp.ramMiBMax
	}

	return rc
}

// Copied from cpu.TimesStat.Total(), since that func is deprecated.
func totalCPU(c *cpu.TimesStat) float64 {
	total := c.User + c.System + c.Idle + c.Nice + c.Iowait + c.Irq +
		c.Softirq + c.Steal + c.Guest + c.GuestNice

	return total
}
