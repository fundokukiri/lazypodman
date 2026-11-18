package panels

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListPanelSelectionStopsAtTheEndsByDefault(t *testing.T) {
	panel := &ListPanel[int]{List: NewFilteredList[int]()}
	panel.List.SetItems([]int{1, 2, 3})
	panel.SetSelectedLineIdx(2)

	panel.SelectNextLine()
	assert.Equal(t, 2, panel.SelectedIdx)

	panel.SetSelectedLineIdx(0)
	panel.SelectPrevLine()
	assert.Equal(t, 0, panel.SelectedIdx)
}

func TestListPanelSelectionWrapsWhenEnabled(t *testing.T) {
	panel := &ListPanel[int]{List: NewFilteredList[int](), WrapSelection: true}
	panel.List.SetItems([]int{1, 2, 3})
	panel.SetSelectedLineIdx(2)

	panel.SelectNextLine()
	assert.Equal(t, 0, panel.SelectedIdx)

	panel.SelectPrevLine()
	assert.Equal(t, 2, panel.SelectedIdx)
}

func TestListPanelWrapsAcrossMoreThanOneStep(t *testing.T) {
	panel := &ListPanel[int]{List: NewFilteredList[int](), WrapSelection: true}
	panel.List.SetItems([]int{1, 2, 3})
	panel.SetSelectedLineIdx(0)

	panel.moveSelectedLine(-4)
	assert.Equal(t, 2, panel.SelectedIdx)

	panel.moveSelectedLine(7)
	assert.Equal(t, 0, panel.SelectedIdx)
}
