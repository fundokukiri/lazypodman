# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Kube menu** (`g` on the pods panel): generate a Kubernetes YAML file from a
  pod, or play/down a manifest from `gui.kubeDir` (default
  `<user config dir>/lazypodman/kube`). Generated files are written with mode
  `0600` after a confirmation that names the full path and warns that env vars are
  plaintext; play/down always confirm with the full path and stream podman's output.
  Commands are probed lazily with a fallback to the older `podman generate kube` /
  `podman play kube` syntax. `--replace` and `--build` are not passed.
- **Health** tab on the containers panel (after **Stats**) showing health
  status, failing streak and the recorded probes, all from inspect data the
  socket already provides.
- `t` on the containers panel runs `podman healthcheck run` once. Healthy and
  unhealthy are results shown on the status line (an unhealthy exit is not an
  error); only spawn failures and undocumented exit codes open the error panel.
  Documented exit codes beyond `0` are unverified; anything outside `0`/`1`/`2`
  is treated as an error.
- Per-feature capability state (`unknown`/`available`/`unavailable`): features
  that need the CLI are probed lazily on first use, so startup still runs two
  podman processes. A failed probe adds `health run off: <reason>` to the
  bottom-right corner; a CLI failure shows a single `cli off: <reason>`.
- Pods panel (between Services and Containers) with **Logs**, **Stats**,
  **Containers** and **Config** tabs. Pod logs are merged and prefixed with the
  container name; pod stats are the sum over the pod's containers.
- Pod lifecycle keys: `s` stop, `S` start, `r` restart, `p` pause/unpause,
  `d` remove (with a force option), `b` bulk commands (prune stopped pods),
  `c` custom `podman` commands on the selected pod.
- `gui.showPodColumn` (default `true`): a pod column in the containers panel,
  shown only when pods are available.
- `gui.pods.enabled` (`auto` | `always` | `never`): how hard to try to enable the
  pods feature. Availability needs the `podman` CLI (>= 3.0) on the same store as
  the socket; otherwise the panel is hidden and the reason is shown as
  `pods off: <reason>` in the bottom-right corner.
- `customCommands.pods` / `bulkCommands.pods` config groups, with `{{ .Pod }}`
  template data.

### Changed

- Panel focus keys are derived from one table: with pods enabled the keys are
  Project `1`, Services `2`, Pods `3`, Containers `4`, Images `5`, Volumes `6`,
  Networks `7`; without pods they stay `1`…`6` with no gaps.

## [Fork]

This project started as a Podman-only fork of
[lazydocker](https://github.com/jesseduffield/lazydocker). The original MIT
license and copyright notice are kept in [LICENSE](LICENSE).
