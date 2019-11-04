package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs"
    "log"
    "math"
	"os"
	"os/exec"
	"os/signal"
	"strings"
    "syscall"
    "time"
)

const defaultSubsystems = "cpuacct,memory"
const exitCodeCpuLimitExceeded = 111
const exitCodeEmptyArgs = 2

var errLogger = log.New(os.Stderr, "docker-cgroups-stats: ", log.LstdFlags | log.LUTC)

func printErr(msg string, args ...interface{}) {
    errLogger.Printf(msg+"\n", args...)
}

func getCgroupMountpoints(subsystems []string) (map[string]string, error) {
	subsystemsToPaths := make(map[string]string)

	for _, name := range subsystems {
		path, err := cgroups.FindCgroupMountpoint(name)
        if err != nil {
            return nil, err
        }

		subsystemsToPaths[name] = path
	}

	return subsystemsToPaths, nil
}

func getCgroupsStats(subsystems []string) (*cgroups.Stats, error) {
    mountpoints, err := getCgroupMountpoints(subsystems)
    if err != nil {
        return nil, err
    }

	manager := fs.Manager{Paths: mountpoints}

	stats, err := manager.GetStats()
	if err != nil {
        return nil, err
	}

	return stats, nil
}

func isCpuLimitExceeded(state *os.ProcessState, cpuLimit uint64) bool {
    userTime := state.UserTime()
    systemTime := state.SystemTime()
    total := time.Duration(userTime.Nanoseconds() + systemTime.Nanoseconds())
    cpuLimitSec := time.Duration(cpuLimit) * time.Second

    if total.Round(time.Duration(time.Second)) >= cpuLimitSec {
        return true
    }

    return false
}

func runSubprocess(args []string) (*os.ProcessState, int) {
	subprocess := exec.Command(args[0], args[1:]...)
	subprocess.Stdin = os.Stdin
	subprocess.Stdout = os.Stdout
	subprocess.Stderr = os.Stderr

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan)
	go func(c chan os.Signal) {
		for sig := range c {
			subprocess.Process.Signal(sig)
		}
	}(signalChan)

    err := subprocess.Run()
    state := subprocess.ProcessState
    if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return state, exitError.ExitCode()
		}

        return state, 1
    }

	return state, 0
}

func setCpuLimit(limitSec uint64) error {
    var limit = syscall.Rlimit{Cur: limitSec, Max: limitSec}
    return syscall.Setrlimit(syscall.RLIMIT_CPU, &limit)
}

func writeStats(stats *cgroups.Stats, outputPath string) error {
	outputFile, err := os.Create(outputPath)
	if err != nil {
        return err
	}
	defer outputFile.Close()

	statsJson, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
        return err
	}

	_, err = outputFile.Write(statsJson)
    return err
}

func main() {
	outputPath := flag.String("o", "/golem/stats/cgroups_stats.json", "path to output file")
	subsystems := flag.String("s", defaultSubsystems,
		"cgroup subsystems to be included in the stats (as comma-separated strings)")
    cpuLimit := flag.Uint64("b", math.MaxUint64, "CPU usage limit for the subprocess (in seconds)")

    if len(os.Args) == 1 {
        fmt.Fprintf(os.Stderr, "Usage:\n")
        flag.PrintDefaults()
        os.Exit(exitCodeEmptyArgs)
    }

	flag.Parse()

    err := setCpuLimit(*cpuLimit)
    if err != nil {
        printErr("Setting CPU limit failed. %s", err)
    }
	state, exitCode := runSubprocess(flag.Args())

    stats, err := getCgroupsStats(strings.Split(*subsystems, ","))
    if err != nil {
        printErr("Could not obtain cgroups stats. %s", err)
    } else {
        err := writeStats(stats, *outputPath)
        if err != nil {
            printErr("Writing stats file failed. %s", err)
        }
    }

    if exitCode != 0 && isCpuLimitExceeded(state, *cpuLimit) {
        printErr("CPU limit exceeded.")
        os.Exit(exitCodeCpuLimitExceeded)
    }

    os.Exit(exitCode)
}
