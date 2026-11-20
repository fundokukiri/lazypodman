package gui

import "bytes"

// transcriptDetector watches a log stream for terminal control sequences that
// rarely appear outside a full-screen program. A couple of cursor/erase
// sequences are enough to switch the Logs tab to transcript mode automatically.
type transcriptDetector struct {
	tail      []byte
	triggered bool
	onDetect  func()
}

func newTranscriptDetector(onDetect func()) *transcriptDetector {
	return &transcriptDetector{onDetect: onDetect}
}

func (d *transcriptDetector) Write(p []byte) (int, error) {
	if d == nil || d.triggered {
		return len(p), nil
	}

	buf := make([]byte, 0, len(d.tail)+len(p))
	buf = append(buf, d.tail...)
	buf = append(buf, p...)

	if looksLikeTranscript(buf) {
		d.triggered = true
		if d.onDetect != nil {
			go d.onDetect()
		}
	}

	// Keep enough overlap to recognise a sequence split across two reads.
	if len(buf) > 32 {
		buf = buf[len(buf)-32:]
	}
	d.tail = append(d.tail[:0], buf...)

	return len(p), nil
}

func looksLikeTranscript(data []byte) bool {
	// Entering the alternate screen or turning on synchronized output is a
	// strong signal on its own.
	for _, marker := range [][]byte{
		[]byte("\x1b[?1049h"),
		[]byte("\x1b[?1047h"),
		[]byte("\x1b[?47h"),
		[]byte("\x1b[?2026h"),
		[]byte("\x1b[?25l"),
		[]byte("\x1b[?2004h"),
		[]byte("\x1b[?1000h"),
		[]byte("\x1b[?7l"),
	} {
		if bytes.Contains(data, marker) {
			return true
		}
	}

	score := 0
	for i := 0; i < len(data); i++ {
		if data[i] != 0x1b || i+1 >= len(data) || data[i+1] != '[' {
			continue
		}

		j := i + 2
		for j < len(data) && data[j] >= 0x20 && data[j] <= 0x3f {
			j++
		}
		if j >= len(data) {
			break
		}

		switch data[j] {
		case 'H', 'f', 'A', 'B', 'C', 'D', 'J', 'K':
			score++
			if score >= 3 {
				return true
			}
		}

		i = j
	}

	return false
}
