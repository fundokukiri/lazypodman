package gui

import (
	"testing"

	"github.com/jesseduffield/gocui"
	"github.com/stretchr/testify/assert"
)

func TestTerminalInputBytesForwardsVietnameseRunes(t *testing.T) {
	data, ok := terminalInputBytes(0, 'ế', gocui.ModNone)

	assert.True(t, ok)
	assert.Equal(t, []byte("ế"), data)
}

func TestTerminalInputBytesRejectsModifiersAndControlKeys(t *testing.T) {
	if _, ok := terminalInputBytes(0, 'ế', gocui.ModAlt); ok {
		t.Fatal("alt-modified rune must not be typed")
	}

	if _, ok := terminalInputBytes(gocui.KeyCtrlE, 0, gocui.ModNone); ok {
		t.Fatal("control keys must keep their bindings")
	}
}
