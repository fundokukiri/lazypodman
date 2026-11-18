# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

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
