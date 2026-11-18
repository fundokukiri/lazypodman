package gui

import (
	"testing"

	"github.com/jesseduffield/gocui"
	"github.com/stretchr/testify/assert"
)

func TestPaletteItemsCoverPanelsAndActions(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())
	gui.Views.Pods = &gocui.View{}

	items := gui.paletteItems()

	assert.Len(t, items, len(gui.sidePanelKeys())+5)

	labels := map[string]bool{}
	for _, item := range items {
		assert.NotEmpty(t, item.LabelColumns[0])
		assert.False(t, labels[item.LabelColumns[0]], "duplicate palette label: "+item.LabelColumns[0])
		labels[item.LabelColumns[0]] = true
		assert.NotNil(t, item.OnPress)
	}

	assert.True(t, labels["Command palette"] || labels["focus containers panel"])
}

func TestPaletteKeyIsBoundOnce(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())

	count := 0
	for _, binding := range gui.staticKeybindings() {
		if binding.Key == gocui.KeyCtrlP {
			count++
		}
	}

	assert.Equal(t, 1, count)
}

func TestMultiSelectKeysAreBoundOnce(t *testing.T) {
	gui := newKubeGui(&fakeHealthPodClient{}, t.TempDir())

	counts := map[string]int{}
	for _, binding := range gui.staticKeybindings() {
		if binding.ViewName == "containers" {
			if key, ok := binding.Key.(rune); ok {
				counts[string(key)]++
			}

			if binding.Key == gocui.KeySpace {
				counts["space"]++
			}
		}
	}

	assert.Equal(t, 1, counts["space"], "space toggles the mark exactly once")
	assert.Equal(t, 1, counts["Q"], "Q exports exactly once")
}
