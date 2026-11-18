# Run Container Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run a new Podman container from the selected image through a fully editable menu.

**Architecture:** A schema drives both menu rows and argv construction. A `runContainerState` holds text values, toggles and lists. Menu handlers edit that state and reopen the prior menu. Execution builds an argv slice and runs `podman run` without a shell.

**Tech Stack:** Go, gocui menu/prompt panels, existing CLI wrapper, `mgutz/str` argv splitter.

**Spec:** `docs/superpowers/specs/2026-11-18-run-container-design.md`

## Global Constraints

- Podman only, rootless first.
- No new dependencies.
- New strings only in `pkg/i18n/english.go`.
- Never build a shell command for execution.
- Defaults must run the selected image with `-d` and nothing else.
- Menus must never require a new conflicting keybinding.

## Review Focus

- An empty/extra field must not produce an empty argv element.
- Command words and extra args must be split safely and placed on the correct side of the image.
- List edits must not drop or duplicate entries.
- A failed `podman run` must show Podman's output, not only `exit status 125`.
- The `r` binding and its menu item must appear for Images and nowhere else.

---

### Task 1: Schema and argv builder

**Files:**
- Create: `pkg/gui/run_container_config.go`
- Test: `pkg/gui/run_container_config_test.go`

**Interfaces:**
- Produces: `type runContainerState`, `newRunContainerState(image string)`, `(*runContainerState).argv(extraArgs string) []string`, `runContainerFields`, `runContainerGroups`.

- [ ] Write tests for defaults, repeated list flags, positional command, extra args and text escaping.
- [ ] Add the schema and builder until tests pass.
- [ ] Commit.

### Task 2: Menus and edits

**Files:**
- Create: `pkg/gui/run_container_menu.go`
- Modify: `pkg/gui/confirmation_panel.go` (prompt with initial value)
- Test: `pkg/gui/run_container_menu_test.go`

**Interfaces:**
- Consumes: `runContainerState`.
- Produces: `handleImagesRunContainer`, `showRunContainerMenu`, `showRunContainerGroup`, `showRunContainerList`, `editRunField`.

- [ ] Write tests for row ordering, list add/remove and value labels.
- [ ] Implement root, advanced, choice and list menus.
- [ ] Implement prompt editing with an initial value.
- [ ] Commit.

### Task 3: Preview and execution

**Files:**
- Modify: `pkg/gui/run_container_menu.go`
- Test: `pkg/gui/run_container_menu_test.go`

**Interfaces:**
- Consumes: `argv`.
- Produces: `previewRunContainer`, `executeRunContainer`.

- [ ] Write tests for the preview string and failure text.
- [ ] Implement confirmation, detached execution and attached execution.
- [ ] Refresh containers/services after success.
- [ ] Commit.

### Task 4: Wiring, i18n, docs

**Files:**
- Modify: `pkg/gui/keybindings.go`, `pkg/i18n/english.go`, `README.md`
- Regenerate: `docs/keybindings/*.md`
- Test: `pkg/gui/keys_test.go`

- [ ] Add the `r` images binding before the existing images bindings.
- [ ] Add strings and confirm the item is first in the images section.
- [ ] Document the menu and fields in README.
- [ ] Run gofmt, vet, tests, lint and cheatsheet check.
- [ ] Commit.
