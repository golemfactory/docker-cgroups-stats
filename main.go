package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs"
	"os"
	"os/exec"
	"os/signal"
	"strings"
)

const defaultSubsystems = "cpuacct,memory"

func exit(errorMsg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, errorMsg+"\n", args...)
	os.Exit(1)
}

func getCgroupMountpoints(subsystems []string) map[string]string {
	subsystemsToPaths := make(map[string]string)

	for _, name := range subsystems {
		path, err := cgroups.FindCgroupMountpoint(name)
		if err != nil {
			exit("Could not find mountpoint for cgroup: %s. %s", name, err)
		}

		subsystemsToPaths[name] = path
	}

	return subsystemsToPaths
}

func getCgroupsStats(subsystems []string) *cgroups.Stats {
	manager := fs.Manager{Paths: getCgroupMountpoints(subsystems)}

	stats, err := manager.GetStats()
	if err != nil {
		exit("Failed to get cgroups stats. %s", err)
	}

	return stats
}

func runSubprocess(args []string) int {
	subprocess := exec.Command(args[0], args[1:]...)
	subprocess.Stdin = os.Stdin
	subprocess.Stdout = os.Stdout
	subprocess.Stderr = os.Stderr

	signalChan := make(chan os.Signal)
	signal.Notify(signalChan)

	go func() {
		for sig := range signalChan {
			subprocess.Process.Signal(sig)
		}
	}()

	if err := subprocess.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode()
		}

		return 1
	}

	return 0
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
	flag.Parse()

	exitCode := runSubprocess(flag.Args())

	stats := getCgroupsStats(strings.Split(*subsystems, ","))
	writeStats(stats, *outputPath)

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
