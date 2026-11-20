# Logs Tab Memory Budget Design

## Intent

A container that writes many or very long log lines can make the Logs tab use
gigabytes of RAM. The tab keeps up to 20,000 gocui view lines until it trims,
and each character costs a cell. Long horizontal lines make that worst case
much easier to reach. Bound the retained text by bytes, keep the newest logs,
and show the initial tail at the bottom without visibly replaying it.

## Goals

- Keep the Logs tab near a small, configurable memory budget.
- Keep following new output after trimming.
- Open at the newest line; the initial fetch stays hidden until it is ready.
- Preserve the existing keys and tabs.

## Non-goals

- Full virtual scrolling over arbitrarily large history.
- Changing the log storage format or adding a dependency.

## Behavior

- New config `logs.maxBytes`, default 1 MiB. `0` disables the byte budget.
- Retain at most `maxBytes` of rendered text and at most 10,000 lines.
- Trim when retained text passes 1.5 times the budget.
- The initial fetch is buffered for a short debounce, then written once with
  the origin at the newest line. No visible top-to-bottom replay.
- Final buffered output is flushed when the stream closes.
- Cancelled tabs discard the buffer and never write into the next tab.

## Approach

Extend `logTrimWriter` with a retained-byte counter and a trim rule that drops
whole oldest lines. Use the same writer to hold the first burst in a bounded
buffer and flush it after 120 ms or on close. This is a focused change: no new
viewer, no keybinding changes, and the existing line cap remains as a safety
limit.

## Defaults

| Setting | Value |
| --- | --- |
| `logs.maxBytes` | `1048576` |
| trim threshold | `1.5 * maxBytes` |
| retained lines | `10000` |
| initial hidden window | `120ms` |
