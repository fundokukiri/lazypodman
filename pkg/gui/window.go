package gui

// func (gui *Gui) currentWindow() string {
// 	// at the moment, we only have one view per window in lazypodman, so we
// 	// are using the view name as the window name
// 	return gui.currentViewName()
// }

// excludes popups
func (gui *Gui) currentStaticWindowName() string {
	name := gui.currentStaticViewName()

	// The terminal lives over the main panel, so layout must treat it as "main".
	if name == "terminal" {
		return "main"
	}

	return name
}

func (gui *Gui) currentSideWindowName() string {
	return gui.currentSideViewName()
}
