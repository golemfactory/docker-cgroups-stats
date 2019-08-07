# docker-cgroups-stats

A wrapper which runs an arbitrary command passed to it and outputs statistics from certain cgroups subsystems once the wrapped program exits.
Cgroups stats are saved in JSON format to a file specified as a command-line argument.

The program is meant to be run inside Docker containers to collect the overall system resource usage of the container's entrypoint.

The cgroups subsystems which are currently included in the command's output are:
- `cpu`
- `cpuacct`
- `cpuset`
- `memory`

## Example usage
```
./docker-cgroups-stats -o /tmp/output.json /bin/sleep 5
```

## Building
```
go build -o docker-cgroups-stats main.go
```

This project uses [Go Modules](https://blog.golang.org/using-go-modules) for dependency management.
This means that dependencies will be resolved and downloaded as part of standard Go commands (in this case `go build`).
Module versions which should be used for the build are listed in `go.mod`.

## Sample output
```
{
  "cpu_stats": {
    "cpu_usage": {
      "total_usage": 210742887,
      "percpu_usage": [
        2294161,
        4289803,
        2629870,
        13575808,
        153013241,
        4415022,
        5868137,
        5475479,
        3252549,
        3329389,
        7260411,
        5347658
      ],
      "usage_in_kernelmode": 30000000,
      "usage_in_usermode": 150000000
    },
    "throttling_data": {}
  },
  "memory_stats": {
    "usage": {
      "usage": 4554752,
      "max_usage": 7172096,
      "failcnt": 0,
      "limit": 9223372036854771712
    },
    "swap_usage": {
      "failcnt": 0,
      "limit": 0
    },
    "kernel_usage": {
      "usage": 2748416,
      "max_usage": 2818048,
      "failcnt": 0,
      "limit": 9223372036854771712
    },
    "kernel_tcp_usage": {
      "failcnt": 0,
      "limit": 9223372036854771712
    },
    "stats": {
      "active_anon": 1495040,
      "active_file": 0,
      "cache": 0,
      "dirty": 0,
      "hierarchical_memory_limit": 9223372036854771712,
      "inactive_anon": 0,
      "inactive_file": 0,
      "mapped_file": 0,
      "pgfault": 3483,
      "pgmajfault": 0,
      "pgpgin": 2574,
      "pgpgout": 2198,
      "rss": 1540096,
      "rss_huge": 0,
      "shmem": 0,
      "total_active_anon": 1495040,
      "total_active_file": 0,
      "total_cache": 0,
      "total_dirty": 0,
      "total_inactive_anon": 0,
      "total_inactive_file": 0,
      "total_mapped_file": 0,
      "total_pgfault": 3483,
      "total_pgmajfault": 0,
      "total_pgpgin": 2574,
      "total_pgpgout": 2198,
      "total_rss": 1540096,
      "total_rss_huge": 0,
      "total_shmem": 0,
      "total_unevictable": 0,
      "total_writeback": 0,
      "unevictable": 0,
      "writeback": 0
    }
  },
  "pids_stats": {},
  "blkio_stats": {}
}
```
