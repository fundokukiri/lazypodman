package gui

import (
	"fmt"

	"github.com/jesseduffield/gocui"
)

// modCtrlShift is tcell.ModCtrl | tcell.ModShift. gocui only exports ModNone
// and ModAlt; terminals with an enhanced keyboard protocol can report this
// combination for ctrl+shift+<letter>.
const modCtrlShift = gocui.Modifier(3)

// Binding - a keybinding mapping a key and modifier to a handler. The keypress
// is only handled if the given view has focus, or handled globally if the view
// is ""
type Binding struct {
	ViewName    string
	Handler     func(*gocui.Gui, *gocui.View) error
	Key         interface{} // FIXME: find out how to get `gocui.Key | rune`
	Modifier    gocui.Modifier
	Description string
}

// GetKey is a function.
func (b *Binding) GetKey() string {
	if b.Key == gocui.KeyF2 {
		return "F2"
	}

	key := 0

	switch b.Key.(type) {
	case rune:
		key = int(b.Key.(rune))
	case gocui.Key:
		key = int(b.Key.(gocui.Key))
	}

	// special keys
	switch key {
	case 27:
		return "esc"
	case 13:
		return "enter"
	case 32:
		return "space"
	case 9:
		return "tab"
	case 65514:
		return "►"
	case 65515:
		return "◄"
	case 65517:
		return "▲"
	case 65516:
		return "▼"
	case 65508:
		return "PgUp"
	case 65507:
		return "PgDn"
	}

	// Control keys would otherwise render as invisible characters in the generated
	// cheatsheets; name them the way a user would type them.
	if key >= 1 && key <= 26 {
		if b.Modifier == modCtrlShift {
			return "ctrl+shift+" + string(rune('a'+key-1))
		}

		return "ctrl+" + string(rune('a'+key-1))
	}

	name := fmt.Sprintf("%c", key)
	if b.Modifier == modCtrlShift {
		name = "ctrl+shift+" + name
	}

	return name
}

// GetInitialKeybindings is a function.
// staticKeybindings is the key table with no GUI state involved, so tests can
// inspect it without building panels and views.
func (gui *Gui) staticKeybindings() []*Binding {
	bindings := []*Binding{
		{
			ViewName: "",
			Key:      gocui.KeyEsc,
			Modifier: gocui.ModNone,
			Handler:  wrappedHandler(gui.escape),
		},
		{
			ViewName: "",
			Key:      'q',
			Modifier: gocui.ModNone,
			Handler:  gui.quit,
		},
		{
			// q is text when the terminal view has focus, so keep a quit key
			// that works there too. The terminal binds the same key locally.
			ViewName: "",
			Key:      gocui.KeyCtrlQ,
			Modifier: gocui.ModNone,
			Handler:  gui.quit,
		},
		{
			ViewName: "",
			Key:      gocui.KeyCtrlC,
			Modifier: gocui.ModNone,
			Handler:  gui.quit,
		},
		{
			ViewName: "",
			Key:      gocui.KeyPgup,
			Modifier: gocui.ModNone,
			Handler:  wrappedHandler(gui.scrollUpMain),
		},
		{
			ViewName: "",
			Key:      gocui.KeyPgdn,
			Modifier: gocui.ModNone,
			Handler:  wrappedHandler(gui.scrollDownMain),
		},
		{
			ViewName: "",
			Key:      gocui.KeyCtrlU,
			Modifier: gocui.ModNone,
			Handler:  wrappedHandler(gui.scrollUpMain),
		},
		{
			ViewName: "",
			Key:      gocui.KeyCtrlD,
			Modifier: gocui.ModNone,
			Handler:  wrappedHandler(gui.scrollDownMain),
		},
		{
			ViewName: "",
			Key:      gocui.KeyEnd,
			Modifier: gocui.ModNone,
			Handler:  gui.autoScrollMain,
		},
		{
			ViewName: "",
			Key:      gocui.KeyHome,
			Modifier: gocui.ModNone,
			Handler:  gui.jumpToTopMain,
		},
		{
			ViewName: "",
			Key:      'x',
			Modifier: gocui.ModNone,
			Handler:  gui.handleCreateOptionsMenu,
		},
		{
			ViewName: "",
			Key:      '?',
			Modifier: gocui.ModNone,
			Handler:  gui.handleCreateOptionsMenu,
		},
		{
			ViewName:    "",
			Key:         gocui.KeyCtrlP,
			Modifier:    gocui.ModNone,
			Handler:     gui.handleCommandPalette,
			Description: gui.Tr.PaletteTitle,
		},
		{
			ViewName: "",
			Key:      'X',
			Modifier: gocui.ModNone,
			Handler:  gui.handleCustomCommand,
		},
		{
			ViewName:    "",
			Key:         'B',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleBuildImage,
			Description: gui.Tr.BuildImage,
		},
		{
			ViewName:    "project",
			Key:         'e',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleEditConfig,
			Description: gui.Tr.EditConfig,
		},
		{
			ViewName:    "project",
			Key:         'o',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleOpenConfig,
			Description: gui.Tr.OpenConfig,
		},
		{
			ViewName:    "project",
			Key:         'm',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleViewAllLogs,
			Description: gui.Tr.ViewLogs,
		},
		{
			ViewName: "menu",
			Key:      gocui.KeyEsc,
			Modifier: gocui.ModNone,
			Handler:  wrappedHandler(gui.handleMenuCancel),
		},
		{
			ViewName: "menu",
			Key:      'q',
			Modifier: gocui.ModNone,
			Handler:  wrappedHandler(gui.handleMenuCancel),
		},
		{
			ViewName: "menu",
			// gocui delivers the spacebar as KeySpace, not the ' ' rune (it clears
			// the rune deliberately so it can match Key instead).
			Key:      gocui.KeySpace,
			Modifier: gocui.ModNone,
			Handler:  wrappedHandler(gui.handleMenuPress),
		},
		{
			ViewName: "menu",
			Key:      gocui.KeyEnter,
			Modifier: gocui.ModNone,
			Handler:  wrappedHandler(gui.handleMenuPress),
		},
		{
			ViewName: "menu",
			Key:      'y',
			Modifier: gocui.ModNone,
			Handler:  wrappedHandler(gui.handleMenuPress),
		},
		{
			ViewName:    "containers",
			Key:         'd',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainersRemoveMenu,
			Description: gui.Tr.Remove,
		},
		{
			ViewName:    "containers",
			Key:         'e',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleHideStoppedContainers,
			Description: gui.Tr.HideStopped,
		},
		{
			ViewName:    "containers",
			Key:         'p',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainerPause,
			Description: gui.Tr.Pause,
		},
		{
			ViewName:    "containers",
			Key:         's',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainerStop,
			Description: gui.Tr.Stop,
		},
		{
			ViewName:    "containers",
			Key:         'r',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainerRestart,
			Description: gui.Tr.Restart,
		},
		{
			ViewName:    "containers",
			Key:         'C',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainersCommitImage,
			Description: gui.Tr.CommitImage,
		},
		{
			ViewName:    "containers",
			Key:         'v',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainerFilesMenu,
			Description: gui.Tr.ContainerFiles,
		},
		{
			ViewName:    "containers",
			Key:         'a',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainerAttach,
			Description: gui.Tr.Attach,
		},
		{
			ViewName:    "containers",
			Key:         'A',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainerAttachCurrentTerminal,
			Description: gui.Tr.AttachCurrentTerminal,
		},
		{
			ViewName:    "containers",
			Key:         gocui.KeyCtrlA,
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainerAttachFullLogs,
			Description: gui.Tr.AttachFullLogs,
		},
		{
			ViewName:    "containers",
			Key:         gocui.KeyF2,
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainerAttachShellTab,
			Description: gui.Tr.AttachShellTab,
		},
		{
			ViewName:    "containers",
			Key:         'm',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainerViewLogs,
			Description: gui.Tr.ViewLogs,
		},
		{
			ViewName:    "containers",
			Key:         't',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainerHealthcheckRun,
			Description: gui.Tr.RunHealthcheck,
		},
		{
			ViewName: "containers",
			// Must be KeySpace: gocui clears the rune for the spacebar, so a rune
			// ' ' binding silently never fires.
			Key:         gocui.KeySpace,
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainersToggleMark,
			Description: gui.Tr.ToggleMark,
		},
		{
			ViewName:    "containers",
			Key:         'Q',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleQuadletMenu,
			Description: gui.Tr.QuadletMenuTitle,
		},
		{
			ViewName:    "containers",
			Key:         gocui.KeyCtrlX,
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainersTerminal,
			Description: gui.Tr.ExecShell,
		},
		{
			ViewName:    "containers",
			Key:         'c',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainersCustomCommand,
			Description: gui.Tr.RunCustomCommand,
		},
		{
			ViewName:    "containers",
			Key:         'b',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainersBulkCommand,
			Description: gui.Tr.ViewBulkCommands,
		},
		{
			ViewName:    "containers",
			Key:         'w',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleContainersOpenInBrowserCommand,
			Description: gui.Tr.OpenInBrowser,
		},
		{
			ViewName:    "services",
			Key:         'u',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServiceUp,
			Description: gui.Tr.UpService,
		},
		{
			ViewName:    "services",
			Key:         'd',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServiceRemoveMenu,
			Description: gui.Tr.RemoveService,
		},
		{
			ViewName:    "services",
			Key:         's',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServiceStop,
			Description: gui.Tr.Stop,
		},
		{
			ViewName:    "services",
			Key:         'p',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServicePause,
			Description: gui.Tr.Pause,
		},
		{
			ViewName:    "services",
			Key:         'r',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServiceRestart,
			Description: gui.Tr.Restart,
		},
		{
			ViewName:    "services",
			Key:         'S',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServiceStart,
			Description: gui.Tr.Start,
		},
		{
			ViewName:    "services",
			Key:         'a',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServiceAttach,
			Description: gui.Tr.Attach,
		},
		{
			ViewName:    "services",
			Key:         'A',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServiceAttachCurrentTerminal,
			Description: gui.Tr.AttachCurrentTerminal,
		},
		{
			ViewName:    "services",
			Key:         gocui.KeyF2,
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServiceAttachShellTab,
			Description: gui.Tr.AttachShellTab,
		},
		{
			ViewName:    "services",
			Key:         'm',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServiceRenderLogsToMain,
			Description: gui.Tr.ViewLogs,
		},
		{
			ViewName:    "services",
			Key:         'U',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleProjectUp,
			Description: gui.Tr.UpProject,
		},
		{
			ViewName:    "services",
			Key:         'D',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleProjectDown,
			Description: gui.Tr.DownProject,
		},
		{
			ViewName:    "services",
			Key:         'R',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServiceRestartMenu,
			Description: gui.Tr.ViewRestartOptions,
		},
		{
			ViewName:    "services",
			Key:         'c',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServicesCustomCommand,
			Description: gui.Tr.RunCustomCommand,
		},
		{
			ViewName:    "services",
			Key:         'b',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServicesBulkCommand,
			Description: gui.Tr.ViewBulkCommands,
		},
		{
			ViewName:    "services",
			Key:         gocui.KeyCtrlX,
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServicesExecShell,
			Description: gui.Tr.ExecShell,
		},
		{
			ViewName:    "services",
			Key:         'w',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleServicesOpenInBrowserCommand,
			Description: gui.Tr.OpenInBrowser,
		},
		{
			ViewName:    "images",
			Key:         'r',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleImagesRunContainer,
			Description: gui.Tr.RunContainer,
		},
		{
			ViewName:    "images",
			Key:         'c',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleImagesCustomCommand,
			Description: gui.Tr.RunCustomCommand,
		},
		{
			ViewName:    "images",
			Key:         'u',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleImagesUpdate,
			Description: gui.Tr.UpdateImage,
		},
		{
			ViewName:    "images",
			Key:         'd',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleImagesRemoveMenu,
			Description: gui.Tr.RemoveImage,
		},
		{
			ViewName:    "images",
			Key:         'b',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleImagesBulkCommand,
			Description: gui.Tr.ViewBulkCommands,
		},
		{
			ViewName:    "volumes",
			Key:         'c',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleVolumesCustomCommand,
			Description: gui.Tr.RunCustomCommand,
		},
		{
			ViewName:    "volumes",
			Key:         'd',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleVolumesRemoveMenu,
			Description: gui.Tr.RemoveVolume,
		},
		{
			ViewName:    "volumes",
			Key:         'b',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleVolumesBulkCommand,
			Description: gui.Tr.ViewBulkCommands,
		},
		{
			ViewName:    "networks",
			Key:         'c',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleNetworksCustomCommand,
			Description: gui.Tr.RunCustomCommand,
		},
		{
			ViewName:    "networks",
			Key:         'd',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleNetworksRemoveMenu,
			Description: gui.Tr.RemoveNetwork,
		},
		{
			ViewName:    "networks",
			Key:         'b',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleNetworksBulkCommand,
			Description: gui.Tr.ViewBulkCommands,
		},
		{
			ViewName: "main",
			Key:      gocui.KeyEnter,
			Modifier: gocui.ModNone,
			Handler:  gui.handleFocusTerminal,
		},
		{
			ViewName:    "main",
			Key:         gocui.KeyCtrlL,
			Modifier:    gocui.ModNone,
			Handler:     gui.handleToggleFullLogs,
			Description: gui.Tr.AllLogs,
		},
		{
			ViewName:    "main",
			Key:         gocui.KeyCtrlT,
			Modifier:    gocui.ModNone,
			Handler:     gui.handleToggleTranscriptLogs,
			Description: gui.Tr.TranscriptLogs,
		},
		{
			ViewName:    "main",
			Key:         gocui.KeyCtrlF,
			Modifier:    gocui.ModNone,
			Handler:     gui.handleLogSearch,
			Description: gui.Tr.LogSearchTitle,
		},
		{
			ViewName:    "main",
			Key:         'G',
			Modifier:    gocui.ModNone,
			Handler:     gui.handleLogsJumpToLatest,
			Description: gui.Tr.JumpToLatest,
		},
		{
			ViewName:    "containers",
			Key:         gocui.KeyCtrlL,
			Modifier:    gocui.ModNone,
			Handler:     gui.handleToggleFullLogs,
			Description: gui.Tr.AllLogs,
		},
		{
			ViewName:    "containers",
			Key:         gocui.KeyCtrlT,
			Modifier:    gocui.ModNone,
			Handler:     gui.handleToggleTranscriptLogs,
			Description: gui.Tr.TranscriptLogs,
		},
		{
			ViewName:    "main",
			Key:         gocui.KeyEsc,
			Modifier:    gocui.ModNone,
			Handler:     gui.handleExitMain,
			Description: gui.Tr.Return,
		},
		{
			ViewName: "main",
			Key:      gocui.KeyArrowLeft,
			Modifier: gocui.ModNone,
			Handler:  gui.scrollLeftMain,
		},
		{
			ViewName: "main",
			Key:      gocui.KeyArrowRight,
			Modifier: gocui.ModNone,
			Handler:  gui.scrollRightMain,
		},
		{
			ViewName: "main",
			Key:      'h',
			Modifier: gocui.ModNone,
			Handler:  gui.scrollLeftMain,
		},
		{
			ViewName: "main",
			Key:      'l',
			Modifier: gocui.ModNone,
			Handler:  gui.scrollRightMain,
		},
		{
			ViewName: "filter",
			Key:      gocui.KeyEnter,
			Modifier: gocui.ModNone,
			Handler:  wrappedHandler(gui.commitFilter),
		},
		{
			ViewName: "filter",
			Key:      gocui.KeyEsc,
			Modifier: gocui.ModNone,
			Handler:  wrappedHandler(gui.escapeFilterPrompt),
		},
		{
			ViewName: "",
			Key:      'J',
			Modifier: gocui.ModNone,
			Handler:  wrappedHandler(gui.scrollDownMain),
		},
		{
			ViewName: "",
			Key:      'K',
			Modifier: gocui.ModNone,
			Handler:  wrappedHandler(gui.scrollUpMain),
		},
		{
			ViewName: "",
			Key:      'H',
			Modifier: gocui.ModNone,
			Handler:  gui.scrollLeftMain,
		},
		{
			ViewName: "",
			Key:      'L',
			Modifier: gocui.ModNone,
			Handler:  gui.scrollRightMain,
		},
		{
			ViewName:    "",
			Key:         '+',
			Handler:     wrappedHandler(gui.nextScreenMode),
			Description: gui.Tr.LcNextScreenMode,
		},
		{
			ViewName:    "",
			Key:         '_',
			Handler:     wrappedHandler(gui.prevScreenMode),
			Description: gui.Tr.LcPrevScreenMode,
		},
	}

	return bindings
}

// GetInitialKeybindings returns the static table plus the per-panel movement and
// click bindings, which need the panels to exist.
func (gui *Gui) GetInitialKeybindings() []*Binding {
	bindings := gui.staticKeybindings()
	bindings = append(bindings, gui.terminalKeyBindings()...)

	for _, panel := range gui.allSidePanels() {
		bindings = append(bindings, []*Binding{
			{ViewName: panel.GetView().Name(), Key: gocui.KeyArrowLeft, Modifier: gocui.ModNone, Handler: gui.previousView},
			{ViewName: panel.GetView().Name(), Key: gocui.KeyArrowRight, Modifier: gocui.ModNone, Handler: gui.nextView},
			{ViewName: panel.GetView().Name(), Key: 'h', Modifier: gocui.ModNone, Handler: gui.previousView},
			{ViewName: panel.GetView().Name(), Key: 'l', Modifier: gocui.ModNone, Handler: gui.nextView},
			{ViewName: panel.GetView().Name(), Key: gocui.KeyTab, Modifier: gocui.ModNone, Handler: gui.nextView},
			{ViewName: panel.GetView().Name(), Key: gocui.KeyBacktab, Modifier: gocui.ModNone, Handler: gui.previousView},
		}...)
	}

	setUpDownClickBindings := func(viewName string, onUp func() error, onDown func() error, onClick func() error) {
		bindings = append(bindings, []*Binding{
			{ViewName: viewName, Key: 'k', Modifier: gocui.ModNone, Handler: wrappedHandler(onUp)},
			{ViewName: viewName, Key: gocui.KeyArrowUp, Modifier: gocui.ModNone, Handler: wrappedHandler(onUp)},
			{ViewName: viewName, Key: gocui.MouseWheelUp, Modifier: gocui.ModNone, Handler: wrappedHandler(onUp)},
			{ViewName: viewName, Key: 'j', Modifier: gocui.ModNone, Handler: wrappedHandler(onDown)},
			{ViewName: viewName, Key: gocui.KeyArrowDown, Modifier: gocui.ModNone, Handler: wrappedHandler(onDown)},
			{ViewName: viewName, Key: gocui.MouseWheelDown, Modifier: gocui.ModNone, Handler: wrappedHandler(onDown)},
			{ViewName: viewName, Key: gocui.MouseLeft, Modifier: gocui.ModNone, Handler: wrappedHandler(onClick)},
		}...)
	}

	for _, panel := range gui.sidePanelKeys() {
		bindings = append(bindings, &Binding{
			Handler:     gui.handleGoTo(panel.view),
			Key:         panel.key,
			Description: panel.description,
		})
	}

	bindings = append(bindings, gui.podBindings()...)

	for _, panel := range gui.allListPanels() {
		setUpDownClickBindings(panel.GetView().Name(), panel.HandlePrevLine, panel.HandleNextLine, panel.HandleClick)
	}

	setUpDownClickBindings("main", gui.scrollUpMain, gui.scrollDownMain, gui.handleMainClick)

	for _, panel := range gui.allSidePanels() {
		bindings = append(bindings,
			&Binding{
				ViewName:    panel.GetView().Name(),
				Key:         gocui.KeyEnter,
				Modifier:    gocui.ModNone,
				Handler:     gui.handleEnterMain,
				Description: gui.Tr.FocusMain,
			},
			&Binding{
				ViewName:    panel.GetView().Name(),
				Key:         '[',
				Modifier:    gocui.ModNone,
				Handler:     wrappedHandler(panel.HandlePrevMainTab),
				Description: gui.Tr.PreviousContext,
			},
			&Binding{
				ViewName:    panel.GetView().Name(),
				Key:         ']',
				Modifier:    gocui.ModNone,
				Handler:     wrappedHandler(panel.HandleNextMainTab),
				Description: gui.Tr.NextContext,
			},
		)
	}

	for _, panel := range gui.allListPanels() {
		if !panel.IsFilterDisabled() {
			bindings = append(bindings, &Binding{
				ViewName:    panel.GetView().Name(),
				Key:         '/',
				Modifier:    gocui.ModNone,
				Handler:     wrappedHandler(gui.handleOpenFilter),
				Description: gui.Tr.LcFilter,
			})
		}
	}

	return bindings
}

func (gui *Gui) keybindings(g *gocui.Gui) error {
	bindings := gui.GetInitialKeybindings()

	for _, binding := range bindings {
		if err := g.SetKeybinding(binding.ViewName, binding.Key, binding.Modifier, binding.Handler); err != nil {
			return err
		}
	}

	if err := g.SetTabClickBinding("main", gui.onMainTabClick); err != nil {
		return err
	}

	return nil
}

func wrappedHandler(f func() error) func(*gocui.Gui, *gocui.View) error {
	return func(g *gocui.Gui, v *gocui.View) error {
		return f()
	}
}

// podBindings are the pod lifecycle keys. They mirror the containers panel, with
// capital S as start (lowercase s is stop) so the muscle memory carries over.
func (gui *Gui) podBindings() []*Binding {
	return []*Binding{
		{ViewName: "pods", Key: 'd', Modifier: gocui.ModNone, Handler: gui.handlePodRemoveMenu, Description: gui.Tr.Remove},
		{ViewName: "pods", Key: 's', Modifier: gocui.ModNone, Handler: gui.handlePodStop, Description: gui.Tr.StopPod},
		{ViewName: "pods", Key: 'S', Modifier: gocui.ModNone, Handler: gui.handlePodStart, Description: gui.Tr.StartPod},
		{ViewName: "pods", Key: 'r', Modifier: gocui.ModNone, Handler: gui.handlePodRestart, Description: gui.Tr.RestartPod},
		{ViewName: "pods", Key: 'p', Modifier: gocui.ModNone, Handler: gui.handlePodPause, Description: gui.Tr.PausePod},
		{ViewName: "pods", Key: 'b', Modifier: gocui.ModNone, Handler: gui.handlePodsBulkCommand, Description: gui.Tr.ViewBulkCommands},
		{ViewName: "pods", Key: 'c', Modifier: gocui.ModNone, Handler: gui.handlePodsCustomCommand, Description: gui.Tr.RunCustomCommand},
		{ViewName: "pods", Key: 'g', Modifier: gocui.ModNone, Handler: gui.handlePodsKubeMenu, Description: gui.Tr.KubeMenuTitle},
	}
}
