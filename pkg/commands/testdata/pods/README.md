# SYNTHETIC / UNVERIFIED podman fixtures

**These files are synthetic.** They were written by hand from the documented
`podman` flags, not captured from a Podman host. Do not quote them as proof that
a JSON shape or log format is what Podman actually emits.

Real captures belong in `testdata/pods/real/<podman-version>/` (for example
`real/5.2.2/pod_ls.json`), and the parser tests read whatever is present there.
Capture at least:

- `podman pod ls --format json`
- `podman pod inspect <pod>`
- `podman ps -a --format json`
- `podman pod logs --names <pod>` (and the same without `--names`)
- `podman logs --names <container>`

Then re-run `GOFLAGS=-mod=vendor go test ./pkg/commands/ -run 'TestParsePods|TestParseContainerPodMap'`
and report the result as "verified by human (podman <version>)".

Every fixture object carries `"_synthetic": true` so its origin is obvious in a
diff; the parsers must ignore it (that is part of the tolerance test).
