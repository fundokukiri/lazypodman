package gui

import (
	"math"

	"github.com/jesseduffield/gocui"
)

func (gui *Gui) scrollUpMain() error {
	mainView := gui.Views.Main
	mainView.Autoscroll = false
	ox, oy := mainView.Origin()
	newOy := int(math.Max(0, float64(oy-gui.Config.UserConfig.Gui.ScrollHeight)))
	return mainView.SetOrigin(ox, newOy)
}

func (gui *Gui) scrollDownMain() error {
	mainView := gui.Views.Main
	mainView.Autoscroll = false
	ox, oy := mainView.Origin()

	reservedLines := 0
	if !gui.Config.UserConfig.Gui.ScrollPastBottom {
		_, sizeY := mainView.Size()
		reservedLines = sizeY
	}

	totalLines := mainView.ViewLinesHeight()
	if oy+reservedLines >= totalLines {
		return nil
	}

	return mainView.SetOrigin(ox, oy+gui.Config.UserConfig.Gui.ScrollHeight)
}

func (gui *Gui) scrollLeftMain(g *gocui.Gui, v *gocui.View) error {
	mainView := gui.Views.Main
	ox, oy := mainView.Origin()
	newOx := int(math.Max(0, float64(ox-gui.Config.UserConfig.Gui.ScrollHeight)))

	return mainView.SetOrigin(newOx, oy)
}

func (gui *Gui) scrollRightMain(g *gocui.Gui, v *gocui.View) error {
	mainView := gui.Views.Main
	ox, oy := mainView.Origin()

	content := mainView.ViewBufferLines()
	var largestNumberOfCharacters int
	for _, txt := range content {
		if len(txt) > largestNumberOfCharacters {
			largestNumberOfCharacters = len(txt)
		}
	}

	sizeX, _ := mainView.Size()
	if ox+sizeX >= largestNumberOfCharacters {
		return nil
	}

	return mainView.SetOrigin(ox+gui.Config.UserConfig.Gui.ScrollHeight, oy)
}

func (gui *Gui) autoScrollMain(g *gocui.Gui, v *gocui.View) error {
	gui.Views.Main.Autoscroll = true
	return nil
}

func (gui *Gui) jumpToTopMain(g *gocui.Gui, v *gocui.View) error {
	gui.Views.Main.Autoscroll = false
	_ = gui.Views.Main.SetOrigin(0, 0)
	_ = gui.Views.Main.SetCursor(0, 0)
	return nil
}

func (gui *Gui) onMainTabClick(tabIndex int) error {
	// The main view can already have focus (for example after clicking its
	// frame), so use the side panel from the view stack rather than currentView.
	currentSidePanel, ok := gui.sidePanelByViewName(gui.currentSideViewName())

	if !ok {
		return nil
	}

	currentSidePanel.SetMainTabIndex(tabIndex)

	// Move the highlight immediately instead of waiting for the panel to queue
	// its render task. Logs can still be draining, but the mouse stays usable.
	mainView := gui.GetMainView()
	mainView.Tabs = currentSidePanel.GetMainTabTitles()
	mainView.TabIndex = tabIndex

	return currentSidePanel.HandleSelect()
}

func (gui *Gui) handleEnterMain(g *gocui.Gui, v *gocui.View) error {
	mainView := gui.Views.Main
	mainView.ParentView = v

	return gui.switchFocus(mainView)
}

func (gui *Gui) handleExitMain(g *gocui.Gui, v *gocui.View) error {
	v.ParentView = nil
	return gui.returnFocus()
}

func (gui *Gui) handleMainClick() error {
	if gui.popupPanelFocused() {
		return nil
	}

	currentView := gui.g.CurrentView()

	if currentView.Name() != "main" {
		gui.Views.Main.ParentView = currentView
	}

	return gui.switchFocus(gui.Views.Main)
}
