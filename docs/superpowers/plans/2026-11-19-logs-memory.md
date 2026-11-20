# Logs Tab Memory Budget Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Bound Logs-tab memory by bytes, trim old lines, and show the initial tail at the bottom without a visible replay.

**Architecture:** Extend `logTrimWriter` with a byte budget and a short hidden initial buffer. The existing streaming, wrapping, cancellation and line cap remain. No new viewer or keybindings.

**Tech Stack:** Go, existing `gocui` view and vendored dependencies.

**Spec:** `docs/superpowers/specs/2026-11-19-logs-memory-design.md`

## Global Constraints

- `GOFLAGS=-mod=vendor` for every Go command.
- Never block the UI goroutine; all log work stays in the log task.
- New user-facing strings only if needed; this change needs none.
- Keep `logs.maxBytes: 0` compatible with the old line-only behavior.
- Update README, `docs/Config.md` and `CHANGELOG.md`.

## Review Focus

- A continuous horizontal line must not stop new output after the cap.
- Switching tabs during the hidden initial buffer must not write into the next tab.
- Trimming must not split UTF-8 or ANSI escape sequences.
- `maxBytes: 0` must not unexpectedly cap the view.
- A stream that ends within the debounce window must still display its tail.

---

### Task 1: Configurable byte budget

**Files:**
- Modify: `pkg/config/app_config.go`
- Modify: `pkg/gui/log_trim.go`
- Modify: `pkg/gui/log_trim_test.go`
- Modify: `pkg/gui/container_log_tail_test.go`

**Interfaces:**
- Produces: `LogsConfig.MaxBytes int` with YAML key `maxBytes`
- Produces: `logTrimWriter.maxBytes int`

- [x] **Step 1: Write the failing default test**

```go
func TestDefaultLogsByteBudget(t *testing.T) {
	assert.Equal(t, 1024*1024, config.GetDefaultConfig().Logs.MaxBytes)
}
```

- [x] **Step 2: Run it and confirm it fails**

Run: `GOFLAGS=-mod=vendor go test ./pkg/gui -run TestDefaultLogsByteBudget -count=1`
Expected: FAIL because `MaxBytes` does not exist or is zero.

- [x] **Step 3: Add `MaxBytes`**

Add `MaxBytes int` with `yaml:"maxBytes,omitempty"` to `LogsConfig`; set the default to `1024 * 1024` in `GetDefaultConfig`.

- [x] **Step 4: Write the failing byte-trim test**

```go
func TestLogTrimWriterTrimsByBytes(t *testing.T) {
	target := &fakeLogTarget{height: 10}
	writer := newLogTrimWriter(target, nil, nil)
	writer.maxBytes = 100

	for i := 0; i < 30; i++ {
		_, err := writer.Write([]byte(fmt.Sprintf("line-%02d-abcdefghij\n", i)))
		assert.NoError(t, err)
	}

	lines := target.BufferLines()
	retained := strings.Join(lines, "\n")
	assert.LessOrEqual(t, len(retained), 100)
	assert.Contains(t, retained, "line-29")
	assert.NotContains(t, retained, "line-00")
}
```

- [x] **Step 5: Implement byte trimming in `logTrimWriter`**

Add `bytes int` and `maxBytes int` fields. After each `view.Write`, add `len(out)` to `bytes`. Trim when `lines > logViewTrimAt` or `maxBytes > 0 && bytes > maxBytes+maxBytes/2`. In `trimLocked`, walk `BufferLines()` from the end, accumulating `len(line)+1`, choose the oldest line that keeps the total within `maxBytes` and the last `logViewKeepLines`; set content, `w.lines`, and `w.bytes` from the joined string.

- [x] **Step 6: Run the focused tests**

Run: `GOFLAGS=-mod=vendor go test ./pkg/gui -run 'LogTrim|LogsByteBudget' -count=1`
Expected: PASS.

- [x] **Step 7: Commit**

```bash
git add pkg/config/app_config.go pkg/gui/log_trim.go pkg/gui/log_trim_test.go pkg/gui/container_log_tail_test.go
git commit -m "feat(logs): add a byte budget for the logs view"
```

### Task 2: Hidden initial load

**Files:**
- Modify: `pkg/gui/log_trim.go`
- Modify: `pkg/gui/container_logs.go`
- Modify: `pkg/gui/log_trim_test.go`

**Interfaces:**
- Consumes: `logTrimWriter.maxBytes`
- Produces: `func (w *logTrimWriter) holdInitialFor(delay time.Duration)`
- Produces: `func (w *logTrimWriter) Close()`

- [x] **Step 1: Write the failing hidden-buffer test**

```go
func TestLogTrimWriterHoldsInitialOutput(t *testing.T) {
	target := &fakeLogTarget{height: 5}
	writer := newLogTrimWriter(target, nil, nil)
	writer.holdInitialFor(time.Hour)
	defer writer.Close()

	_, err := writer.Write([]byte("one\ntwo\nthree\n"))
	assert.NoError(t, err)
	assert.Empty(t, target.content, "initial output stays hidden")

	writer.flushInitialForTest()
	assert.Equal(t, "one\ntwo\nthree\n", target.content)
	assert.Equal(t, 1, target.originY)
}
```

Add `TestLogTrimWriterCloseFlushesInitial` and a stale-buffer test using a callback that returns true.

- [x] **Step 2: Run the tests and confirm they fail**

Run: `GOFLAGS=-mod=vendor go test ./pkg/gui -run 'Initial' -count=1`
Expected: FAIL because the methods do not exist.

- [x] **Step 3: Implement the initial buffer**

Add `initial []byte`, `initialTimer *time.Timer`, `flushed bool`, `closed bool`. In `Write`, while `!flushed`, wrap output into `initial` (bounded to `maxBytes`, or 1 MiB when `maxBytes == 0`) instead of the view. `holdInitialFor` starts `time.AfterFunc(delay, w.flushInitial)`. `flushInitial` locks, returns when flushed or stale, writes the buffer once, updates `bytes`/`lines`, trims and pins to bottom. `Close` stops the timer and flushes remaining data.

- [x] **Step 4: Wire it into the Logs tab**

In `renderContainerLogsToMainAux`, set `logWriter.maxBytes`, call `logWriter.holdInitialFor(120 * time.Millisecond)`, and `defer logWriter.Close()`.

- [x] **Step 5: Run the focused tests**

Run: `GOFLAGS=-mod=vendor go test ./pkg/gui -run 'LogTrim|Initial' -count=1`
Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add pkg/gui/log_trim.go pkg/gui/container_logs.go pkg/gui/log_trim_test.go
git commit -m "feat(logs): open at the newest line with a hidden initial load"
```

### Task 3: Documentation and full verification

**Files:**
- Modify: `README.md`
- Modify: `docs/Config.md`
- Modify: `CHANGELOG.md`

- [x] **Step 1: Document `logs.maxBytes`**

Add `maxBytes: 1048576` with a note that `0` disables the byte budget and the initial fetch is shown at the bottom.

- [x] **Step 2: Run the full gate**

Run:
`gofmt -s -l main.go pkg/ scripts/`
`GOFLAGS=-mod=vendor go vet ./pkg/...`
`GOFLAGS=-mod=vendor go test ./...`
`bash ./test.sh`
`golangci-lint run ./pkg/...`
`go run scripts/cheatsheet/main.go check`
`GOOS=windows GOARCH=amd64 GOFLAGS=-mod=vendor go build ./...`

Expected: no output from gofmt, all tests pass, lint exits 0.

- [x] **Step 3: Commit**

```bash
git add README.md docs/Config.md CHANGELOG.md
git commit -m "docs(logs): document the logs memory budget"
```

## Self-Review

- Spec coverage: byte budget (Task 1), hidden initial tail (Task 2), docs and gate (Task 3).
- No new strings or keybindings.
- Cancellation is checked before the initial flush.
- `maxBytes: 0` keeps the old line-only trim path.
