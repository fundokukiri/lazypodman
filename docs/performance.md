# Startup performance

This page records how startup cost is measured and what the current numbers are.
The measurements are **synthetic**: a fake `podman` script with a fixed delay,
because the numbers of interest are "how many podman processes does startup spawn,
and do they run in parallel". A real rootless podman adds its own storage init per
process, which is exactly why the process count matters.

## What startup does

| Step | Cost |
| --- | --- |
| socket client + `podman info --format` | one podman process (version + socket in one call) |
| compose check (`podman compose version`) | one process, runs concurrently with the CLI check |
| first refresh: containers/services, images, volumes, networks | socket HTTP calls, one goroutine each |
| first refresh: pods | two CLI calls, started before the socket refresh |
| healthcheck / kube / `pod logs --help` | **not** at startup: probed lazily on first use |

So the startup critical path is:

```
max(socket refresh, pod refresh)  after  max(cli check, compose check)
```

Before the Podman-native phases the `.container`/kube/health work could add a third
sequential CLI spawn; the healthcheck and kube probes are lazy by design
(`Capabilities` starts as `Unknown`), and the last remaining startup-only probe
(`pod logs --help`) is now lazy too.

## How to measure

```sh
# 300 ms per spawned podman process, 3 iterations
BENCH_PODMAN_DELAY_MS=300 GOFLAGS=-mod=vendor \
  go test ./pkg/commands/ -run '^$' \
  -bench 'BenchmarkStartupCLICheck|BenchmarkRefreshPodState' -benchtime=3x
```

The two guard tests run on every `go test ./...`:

- `TestStartupCLICheckSpawnsOneProcess` — fails if the enablement check ever spawns a
  second process again.
- `TestRefreshPodStateRunsItsTwoCallsConcurrently` — fails if `pod ls` and
  `ps -a --format json` stop overlapping (bound: 150 ms delay each, must finish in
  under 300 ms).

Both use a POSIX shell fake, so they are skipped on Windows only by the same
mechanism as the other CLI tests (they do not need podman installed).

## Measured numbers

Fake delay 300 ms per process, Linux, Go benchmark, no podman installed:

| Measurement | Before | After |
| --- | --- | --- |
| CLI enablement check (`newCLIPodmanClient`) | 610 ms (2 sequential spawns) | **305 ms** (1 spawn) |
| Pod refresh (`RefreshPodState`) | ~2 spawns sequential | **304 ms** (2 overlapping spawns) |
| Processes spawned by startup | 4 | **3** (1 + 2), the pod pair overlapping the socket refresh |

The first row is a direct before/after measurement of the same benchmark harness; the
second row's "before" is derived from the two 300 ms calls it used to run in sequence.
These numbers are **not** a claim about real podman: a real rootless start is roughly
200–600 ms per process depending on the machine and storage driver, which is why one
spawn instead of two matters more than any micro-optimisation in the TUI.

## Known remaining costs

- The pod refresh still needs two CLI calls because the Docker-compatible API has no
  pod endpoint. `podman ps --format json` includes pod fields on recent versions, but
  that shape is unverified, so it is not used to drop a call.
- The compose check shells out through the user's `commandTemplates`, so a slow
  compose provider delays the compose/services panels, not the container list.
- `pod logs --help` now costs one spawn the first time a Pod logs tab is opened.
