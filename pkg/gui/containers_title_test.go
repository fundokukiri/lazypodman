package gui

import (
	"testing"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/stretchr/testify/assert"
)

func TestContainerTerminalTitle(t *testing.T) {
	ctr := &commands.Container{Name: "web"}
	ctr.ID = "abcdef0123456789"

	assert.Equal(t, "Container: web - abcdef012345", containerTerminalTitle(ctr, false))
	assert.Equal(t, "Container: web - abcdef012345 - tiến trình đang chạy", containerTerminalTitle(ctr, true))
}
