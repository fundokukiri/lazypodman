package panels

import (
	"github.com/jesseduffield/gocui"
	lcUtils "github.com/jesseduffield/lazycore/pkg/utils"
)

type ListPanel[T comparable] struct {
	SelectedIdx int
	List        *FilteredList[T]
	View        *gocui.View
	// WrapSelection makes up/down move from the last item to the first and
	// back, instead of stopping at the ends. Used by menus.
	WrapSelection bool
}

func (self *ListPanel[T]) SetSelectedLineIdx(value int) {
	clampedValue := 0
	if self.List.Len() > 0 {
		clampedValue = lcUtils.Clamp(value, 0, self.List.Len()-1)
	}

	self.SelectedIdx = clampedValue
}

func (self *ListPanel[T]) clampSelectedLineIdx() {
	clamped := lcUtils.Clamp(self.SelectedIdx, 0, self.List.Len()-1)

	if clamped != self.SelectedIdx {
		self.SelectedIdx = clamped
	}
}

// moves the cursor up or down by the given amount (up for negative values)
func (self *ListPanel[T]) moveSelectedLine(delta int) {
	if self.WrapSelection && self.List.Len() > 0 {
		next := (self.SelectedIdx + delta) % self.List.Len()
		if next < 0 {
			next += self.List.Len()
		}
		self.SetSelectedLineIdx(next)
		return
	}

	self.SetSelectedLineIdx(self.SelectedIdx + delta)
}

func (self *ListPanel[T]) SelectNextLine() {
	self.moveSelectedLine(1)
}

func (self *ListPanel[T]) SelectPrevLine() {
	self.moveSelectedLine(-1)
}
