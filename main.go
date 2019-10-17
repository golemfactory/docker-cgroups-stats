package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs"
    "math"
	"os"
	"os/exec"
	"os/signal"
	"strings"
    "syscall"
    "time"
)

const defaultSubsystems = "cpuacct,memory"

func exit(errorMsg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, errorMsg+"\n", args...)
	os.Exit(1)
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

func isBudgetExceeded(state *os.ProcessState, budget uint64) bool {
    userTime := state.UserTime()
    systemTime := state.SystemTime()
    total := time.Duration(userTime.Nanoseconds() + systemTime.Nanoseconds())
    budgetSec := time.Duration(budget) * time.Second

    if total.Round(time.Duration(time.Second)) >= budgetSec {
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

func setCpuLimit(limitSec uint64) {
    var limit = syscall.Rlimit{Cur: limitSec, Max: limitSec}
    err := syscall.Setrlimit(syscall.RLIMIT_CPU, &limit)
    if err != nil {
        exit("Failed setting CPU limit. %s", err)
    }
}

func writeStats(stats *cgroups.Stats, outputPath string) {
	outputFile, err := os.Create(outputPath)
	if err != nil {
		exit("Could not create output file. %s", err)
	}
	defer outputFile.Close()

	statsJson, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		exit("Failed to serialize stats. %s", err)
	}

	_, err = outputFile.Write(statsJson)
	if err != nil {
		exit("Failed writing stats to output file. %s", err)
	}
}

func main() {
	outputPath := flag.String("o", "/golem/stats/cgroups_stats.json", "path to output file")
	subsystems := flag.String("s", defaultSubsystems,
		"cgroup subsystems to be included in the stats (as comma-separated strings)")
    useCgroups := flag.Bool("cgroups", false, "should the program's output include stats from cgroups")
    cpuBudget := flag.Uint64("b", math.MaxUint64, "CPU budget for the subprocess (in seconds)")
	flag.Parse()

    setCpuLimit(*cpuBudget)
	state, exitCode := runSubprocess(flag.Args())

    if exitCode != 0 && isBudgetExceeded(state, *cpuBudget) {
        // TODO report budget exceeded
        fmt.Println("budget exceeded")
    }

    if *useCgroups {
        stats, err := getCgroupsStats(strings.Split(*subsystems, ","))
        if err != nil {
            fmt.Fprintf(os.Stderr, "Could not obtain cgroups stats. %s\n", err)
        } else {
            writeStats(stats, *outputPath)
        }
    }

    os.Exit(exitCode)
}
