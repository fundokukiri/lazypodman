package gui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/fatih/color"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/config"
	"github.com/fundokukiri/lazypodman/pkg/gui/panels"
	"github.com/fundokukiri/lazypodman/pkg/gui/presentation"
	"github.com/fundokukiri/lazypodman/pkg/gui/types"
	"github.com/fundokukiri/lazypodman/pkg/tasks"
	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/jesseduffield/gocui"
	"github.com/samber/lo"
)

func (gui *Gui) getContainersPanel() *panels.SideListPanel[*commands.Container] {
	// Standalone containers are containers which are either one-off containers, or whose service is not part of this podman-compose context.
	isStandaloneContainer := func(container *commands.Container) bool {
		if container.OneOff || container.ServiceName == "" {
			return true
		}

		return !lo.SomeBy(gui.Panels.Services.List.GetAllItems(), func(service *commands.Service) bool {
			return service.Name == container.ServiceName && service.ProjectName == container.ProjectName
		})
	}

	return &panels.SideListPanel[*commands.Container]{
		ContextState: &panels.ContextState[*commands.Container]{
			GetMainTabs: func() []panels.MainTab[*commands.Container] {
				tabs := []panels.MainTab[*commands.Container]{
					{
						Key:    "logs",
						Title:  gui.Tr.LogsTitle,
						Render: gui.renderContainerLogsToMain,
					},
					{
						Key:    "diagnose",
						Title:  gui.Tr.DiagnoseTitle,
						Render: gui.renderContainerDiagnosis,
					},
					{
						Key:    "stats",
						Title:  gui.Tr.StatsTitle,
						Render: gui.renderContainerStats,
					},
					{
						Key:    "health",
						Title:  gui.Tr.HealthTitle,
						Render: gui.renderContainerHealth,
					},
					{
						Key:    "env",
						Title:  gui.Tr.EnvTitle,
						Render: gui.renderContainerEnv,
					},
					{
						Key:    "config",
						Title:  gui.Tr.ConfigTitle,
						Render: gui.renderContainerConfig,
					},
					{
						Key:    "top",
						Title:  gui.Tr.TopTitle,
						Render: gui.renderContainerTop,
					},
					{
						Key:    "events",
						Title:  gui.Tr.EventsTitle,
						Render: gui.renderContainerEvents,
					},
				}

				if gui.Panels.Containers != nil {
					if container, err := gui.Panels.Containers.GetSelectedItem(); err == nil && gui.terminalTabFor(container) {
						// The shell tab only exists after x -> exec shell, right after Logs.
						tabs = append(tabs[:1], append([]panels.MainTab[*commands.Container]{
							{
								Key:    terminalTabKey,
								Title:  gui.Tr.ShellTitle,
								Render: gui.renderContainerTerminal,
							},
						}, tabs[1:]...)...)
					}
				}

				return tabs
			},
			GetItemContextCacheKey: func(container *commands.Container) string {
				// Including the container state in the cache key so that if the container
				// restarts we re-read the logs. In the past we've had some glitchiness
				// where a container restarts but the new logs don't get read.
				// Note that this might be jarring if we have a lot of logs and the container
				// restarts a lot, so let's keep an eye on it.
				return "containers-" + container.ID + "-" + container.Container.State
			},
		},
		ListPanel: panels.ListPanel[*commands.Container]{
			List: panels.NewFilteredList[*commands.Container](),
			View: gui.Views.Containers,
		},
		NoItemsMessage: gui.Tr.NoContainers,
		Gui:            gui.intoInterface(),
		// sortedContainers returns containers sorted by state if c.SortContainersByState is true (follows 1- running, 2- exited, 3- created)
		// and sorted by name if c.SortContainersByState is false
		Sort: func(a *commands.Container, b *commands.Container) bool {
			return sortContainers(a, b, gui.Config.UserConfig.Gui.LegacySortContainers)
		},
		Filter: func(container *commands.Container) bool {
			if !gui.State.ShowExitedContainers && container.Container.State == "exited" {
				return false
			}

			// When project-scoped, apply project and standalone filtering.
			// Otherwise all containers are shown in a flat list regardless
			// of which compose project they belong to.
			if gui.PodmanCommand.IsProjectScoped() {
				// This check must be inside the IsProjectScoped guard: when
				// not project-scoped, services are still derived from container
				// labels, so compose-managed containers from other projects
				// would be incorrectly hidden.
				//
				// Note that this is O(N*M) time complexity where N is the number of services
				// and M is the number of containers. We expect N to be small but M may be large,
				// so we will need to keep an eye on this.
				if !gui.Config.UserConfig.Gui.ShowAllContainers && !isStandaloneContainer(container) {
					return false
				}

				// Filter by selected project. Containers with no project (truly
				// standalone, not from any compose project) are always shown.
				selectedProject := gui.getSelectedProjectName()
				if selectedProject == "" {
					selectedProject = gui.PodmanCommand.LocalProjectName
				}
				if selectedProject != "" && container.ProjectName != "" && container.ProjectName != selectedProject {
					return false
				}
			}

			return true
		},
		GetTableCells: func(container *commands.Container) []string {
			cells := presentation.GetContainerDisplayStrings(&gui.Config.UserConfig.Gui, container, gui.podColumnEnabled())
			// The mark is part of the name cell so the table keeps its width.
			cells[2] = gui.containerMarkPrefix(container.ID) + cells[2]

			return cells
		},
	}
}

var containerStates = map[string]int{
	"running": 1,
	"exited":  2,
	"created": 3,
}

func sortContainers(a *commands.Container, b *commands.Container, legacySort bool) bool {
	if legacySort {
		return a.Name < b.Name
	}

	stateLeft := containerStates[a.Container.State]
	stateRight := containerStates[b.Container.State]
	if stateLeft == stateRight {
		return a.Name < b.Name
	}

	return containerStates[a.Container.State] < containerStates[b.Container.State]
}

func (gui *Gui) renderContainerEnv(container *commands.Container) tasks.TaskFunc {
	return gui.NewSimpleRenderStringTask(func() string { return gui.containerEnv(container) })
}

func (gui *Gui) containerEnv(container *commands.Container) string {
	if !container.DetailsLoaded() {
		return gui.Tr.WaitingForContainerInfo
	}

	if len(container.DetailsSnapshot().Config.Env) == 0 {
		return gui.Tr.NothingToDisplay
	}

	envVarsList := lo.Map(container.DetailsSnapshot().Config.Env, func(envVar string, _ int) []string {
		splitEnv := strings.SplitN(envVar, "=", 2)
		key := splitEnv[0]
		value := ""
		if len(splitEnv) > 1 {
			value = splitEnv[1]
		}
		return []string{
			utils.ColoredString(key+":", color.FgGreen),
			utils.ColoredString(value, color.FgYellow),
		}
	})

	output, err := utils.RenderTable(envVarsList)
	if err != nil {
		gui.Log.Error(err)
		return gui.Tr.CannotDisplayEnvVariables
	}

	return output
}

func (gui *Gui) renderContainerConfig(container *commands.Container) tasks.TaskFunc {
	return gui.NewSimpleRenderStringTask(func() string { return gui.containerConfigStr(container) })
}

func (gui *Gui) containerConfigStr(container *commands.Container) string {
	if !container.DetailsLoaded() {
		return gui.Tr.WaitingForContainerInfo
	}

	padding := 10
	output := ""
	output += utils.WithPadding("ID: ", padding) + container.ID + "\n"
	output += utils.WithPadding("Name: ", padding) + container.Name + "\n"
	output += utils.WithPadding("Image: ", padding) + container.DetailsSnapshot().Config.Image + "\n"
	output += utils.WithPadding("Command: ", padding) + strings.Join(append([]string{container.DetailsSnapshot().Path}, container.DetailsSnapshot().Args...), " ") + "\n"
	output += utils.WithPadding("Labels: ", padding) + utils.FormatMap(padding, container.DetailsSnapshot().Config.Labels)
	output += "\n"

	output += utils.WithPadding("Mounts: ", padding)
	if len(container.DetailsSnapshot().Mounts) > 0 {
		output += "\n"
		for _, mount := range container.DetailsSnapshot().Mounts {
			if mount.Type == "volume" {
				output += fmt.Sprintf("%s%s %s\n", strings.Repeat(" ", padding), utils.ColoredString(string(mount.Type)+":", color.FgYellow), mount.Name)
			} else {
				output += fmt.Sprintf("%s%s %s:%s\n", strings.Repeat(" ", padding), utils.ColoredString(string(mount.Type)+":", color.FgYellow), mount.Source, mount.Destination)
			}
		}
	} else {
		output += "none\n"
	}

	output += utils.WithPadding("Ports: ", padding)
	if len(container.DetailsSnapshot().NetworkSettings.Ports) > 0 {
		output += "\n"
		for k, v := range container.DetailsSnapshot().NetworkSettings.Ports {
			for _, host := range v {
				output += fmt.Sprintf("%s%s %s\n", strings.Repeat(" ", padding), utils.ColoredString(host.HostPort+":", color.FgYellow), k)
			}
		}
	} else {
		output += "none\n"
	}

	details := container.DetailsSnapshot()
	data, err := utils.MarshalIntoYaml(&details)
	if err != nil {
		return fmt.Sprintf("Error marshalling container details: %v", err)
	}

	output += fmt.Sprintf("\nFull details:\n\n%s", utils.ColoredYamlString(string(data)))

	return output
}

// refreshContainerSize asks Podman once, when the Stats tab opens, for the
// container's writable layer and total rootfs size.
func (gui *Gui) refreshContainerSize(ctx context.Context, container *commands.Container) {
	command := exec.CommandContext(ctx, gui.Config.PodmanBinary(),
		"inspect", "--size", "--format", "{{.SizeRw}} {{.SizeRootFs}}", container.ID)
	output, err := command.Output()
	if err != nil {
		gui.Log.Debugf("could not read container size: %v", err)
		return
	}

	fields := strings.Fields(string(output))
	if len(fields) != 2 {
		return
	}
	rw, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return
	}
	rootFs, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return
	}

	container.SetDiskSize(rw, rootFs)
}

func (gui *Gui) renderContainerStats(container *commands.Container) tasks.TaskFunc {
	return gui.NewTickerTask(TickerTaskOpts{
		Func: func(ctx context.Context, notifyStopped chan struct{}) {
			contents, err := presentation.RenderStats(gui.Config.UserConfig, container, gui.Views.Main.Width())
			if err != nil {
				_ = gui.createErrorPanel(err.Error())
			}

			gui.reRenderStringMain(contents)
		},
		Duration: time.Second,
		Before: func(ctx context.Context) {
			gui.clearMainView()
			// `podman inspect --size` can take seconds on a large rootfs, so it
			// must not delay the first stats render. The next tick picks the
			// values up once this finishes (or the line stays unavailable).
			go gui.refreshContainerSize(ctx, container)
		},
		Wrap:       false, // wrapping looks bad here so we're overriding the config value
		Autoscroll: false,
	})
}

func (gui *Gui) renderContainerTop(container *commands.Container) tasks.TaskFunc {
	return gui.NewTickerTask(TickerTaskOpts{
		Func: func(ctx context.Context, notifyStopped chan struct{}) {
			contents, err := container.RenderTop(ctx)
			if err != nil {
				gui.RenderStringMain(err.Error())
			}

			gui.reRenderStringMain(contents)
		},
		Duration: time.Second,
		Before: func(ctx context.Context) {
			gui.clearMainView()
			// `podman inspect --size` can take seconds on a large rootfs, so it
			// must not delay the first stats render. The next tick picks the
			// values up once this finishes (or the line stays unavailable).
			go gui.refreshContainerSize(ctx, container)
		},
		Wrap:       gui.Config.UserConfig.Gui.WrapMainPanel,
		Autoscroll: false,
	})
}

func (gui *Gui) restoreSelectedContainer(selected *commands.Container) {
	if selected == nil {
		return
	}

	for i, container := range gui.Panels.Containers.List.GetItems() {
		if container.ID == selected.ID {
			gui.Panels.Containers.SetSelectedLineIdx(i)
			return
		}
	}
}

func (gui *Gui) refreshContainersAndServices() error {
	if gui.Views.Containers == nil {
		// if the containersView hasn't been instantiated yet we just return
		return nil
	}

	// keep track of current service selected so that we can reposition our cursor if it moves position in the list
	originalSelectedLineIdx := gui.Panels.Services.SelectedIdx
	selectedService, isServiceSelected := gui.Panels.Services.List.TryGet(originalSelectedLineIdx)

	// A restart moves a container between the exited and running groups. If we
	// only keep the row index, the cursor and the main tab jump to whichever
	// container now occupies that row. Keep the selected container by ID.
	originalContainerIdx := gui.Panels.Containers.SelectedIdx
	selectedContainer, isContainerSelected := gui.Panels.Containers.List.TryGet(originalContainerIdx)

	containers, services, err := gui.PodmanCommand.RefreshContainersAndServices(
		gui.Panels.Containers.List.GetAllItems(),
	)
	if err != nil {
		return err
	}

	gui.Panels.Services.SetItems(services)
	gui.Panels.Containers.SetItems(containers)

	if isContainerSelected {
		gui.restoreSelectedContainer(selectedContainer)
	}

	// see if our selected service has moved
	if isServiceSelected {
		for i, service := range gui.Panels.Services.List.GetItems() {
			if service.ID == selectedService.ID {
				if i == originalSelectedLineIdx {
					break
				}
				gui.Panels.Services.SetSelectedLineIdx(i)
				gui.Panels.Services.Refocus()
			}
		}
	}

	return gui.renderContainersAndServices()
}

func (gui *Gui) renderContainersAndServices() error {
	if err := gui.Panels.Services.RerenderList(); err != nil {
		return err
	}

	if err := gui.Panels.Containers.RerenderList(); err != nil {
		return err
	}

	return nil
}

func (gui *Gui) handleHideStoppedContainers(g *gocui.Gui, v *gocui.View) error {
	gui.State.ShowExitedContainers = !gui.State.ShowExitedContainers

	return gui.Panels.Containers.RerenderList()
}

func (gui *Gui) handleContainersRemoveMenu(g *gocui.Gui, v *gocui.View) error {
	targets := gui.containersToActOn()
	if len(targets) == 0 {
		return nil
	}

	removeAll := func(configOptions container.RemoveOptions) error {
		return gui.WithWaitingStatus(gui.Tr.RemovingStatus, func() error {
			for _, ctr := range targets {
				if err := ctr.Remove(configOptions); err != nil {
					if commands.HasErrorCode(err, commands.MustStopContainer) {
						return gui.createConfirmationPanel(gui.Tr.Confirm, gui.Tr.MustForceToRemoveContainer, func(g *gocui.Gui, v *gocui.View) error {
							return gui.WithWaitingStatus(gui.Tr.RemovingStatus, func() error {
								configOptions.Force = true

								return ctr.Remove(configOptions)
							})
						}, nil)
					}

					return gui.createErrorPanel(err.Error())
				}
			}

			gui.clearMarkedContainers()

			return nil
		})
	}

	podman := gui.Config.PodmanBinary()
	menuItems := []*types.MenuItem{
		{
			LabelColumns: []string{gui.Tr.Remove, podman + " rm " + targets[0].ID[1:10]},
			OnPress:      func() error { return removeAll(container.RemoveOptions{}) },
		},
		{
			LabelColumns: []string{gui.Tr.RemoveWithVolumes, podman + " rm --volumes " + targets[0].ID[1:10]},
			OnPress:      func() error { return removeAll(container.RemoveOptions{RemoveVolumes: true}) },
		},
	}

	title := ""
	if len(targets) > 1 {
		title = fmt.Sprintf(gui.Tr.RemoveContainersCount, len(targets))
	}

	return gui.Menu(CreateMenuOptions{
		Title: title,
		Items: menuItems,
	})
}

func (gui *Gui) PauseContainer(container *commands.Container) error {
	details := container.DetailsSnapshot()
	if details.State == nil {
		return gui.createErrorPanel(gui.Tr.WaitingForContainerInfo)
	}

	return gui.WithWaitingStatus(gui.Tr.PausingStatus, func() (err error) {
		if details.State.Paused {
			err = container.Unpause()
		} else {
			err = container.Pause()
		}

		if err != nil {
			return gui.createErrorPanel(err.Error())
		}

		return gui.refreshContainersAndServices()
	})
}

func (gui *Gui) handleContainerPause(g *gocui.Gui, v *gocui.View) error {
	targets := gui.containersToActOn()
	if len(targets) == 0 {
		return nil
	}

	if len(targets) == 1 {
		return gui.PauseContainer(targets[0])
	}

	return gui.WithWaitingStatus(gui.Tr.PausingStatus, func() error {
		for _, ctr := range targets {
			var err error
			if ctr.DetailsSnapshot().State != nil && ctr.DetailsSnapshot().State.Paused {
				err = ctr.Unpause()
			} else {
				err = ctr.Pause()
			}

			if err != nil {
				return gui.createErrorPanel(err.Error())
			}
		}

		gui.clearMarkedContainers()

		return gui.refreshContainersAndServices()
	})
}

func (gui *Gui) handleContainerStop(g *gocui.Gui, v *gocui.View) error {
	targets := gui.containersToActOn()
	if len(targets) == 0 {
		return nil
	}

	prompt := gui.Tr.StopContainer
	if len(targets) > 1 {
		prompt = fmt.Sprintf(gui.Tr.StopContainersCount, len(targets))
	}

	return gui.createConfirmationPanel(gui.Tr.Confirm, prompt, func(g *gocui.Gui, v *gocui.View) error {
		return gui.WithWaitingStatus(gui.Tr.StoppingStatus, func() error {
			for _, ctr := range targets {
				if err := ctr.Stop(); err != nil {
					return gui.createErrorPanel(err.Error())
				}
			}

			gui.clearMarkedContainers()

			return gui.refreshContainersAndServices()
		})
	}, nil)
}

func (gui *Gui) handleContainerRestart(g *gocui.Gui, v *gocui.View) error {
	targets := gui.containersToActOn()
	if len(targets) == 0 {
		return nil
	}

	return gui.WithWaitingStatus(gui.Tr.RestartingStatus, func() error {
		for _, ctr := range targets {
			if err := ctr.Restart(); err != nil {
				return gui.createErrorPanel(err.Error())
			}
		}

		gui.clearMarkedContainers()

		return gui.refreshContainersAndServices()
	})
}

func (gui *Gui) handleContainerAttach(g *gocui.Gui, v *gocui.View) error {
	ctr, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	return gui.openContainerAttachTerminal(ctr)
}

// openContainerAttachTerminal opens the preferred shell in a separate terminal
// window, where the user gets that terminal's own scrollback instead of
// suspending the lazypodman UI. Without a terminal emulator it falls back to
// the current terminal instead of failing.
func (gui *Gui) openContainerAttachTerminal(ctr *commands.Container) error {
	if !containerIsRunning(ctr) {
		return gui.createErrorPanel(gui.Tr.ContainerNotRunning)
	}

	argv := commands.ShellTerminalArgv(gui.Config.PodmanBinary(), ctr.ID)
	if err := gui.OSCommand.OpenExternalTerminal(argv); err != nil {
		if errors.Is(err, commands.ErrNoTerminalEmulator) {
			return gui.attachInCurrentTerminal(ctr)
		}
		return gui.createErrorPanel(err.Error())
	}

	gui.flashStatus(gui.Tr.AttachOpenedInTerminal)

	return nil
}

// attachInCurrentTerminal suspends the UI and runs the preferred shell in the
// current terminal. `A` always uses this path; `a` only uses it as a fallback
// when no external terminal emulator is available.
func (gui *Gui) attachInCurrentTerminal(ctr *commands.Container) error {
	if !containerIsRunning(ctr) {
		return gui.createErrorPanel(gui.Tr.ContainerNotRunning)
	}

	return gui.runAttachCommand(commands.ShellTerminalArgv(gui.Config.PodmanBinary(), ctr.ID))
}

func (gui *Gui) runAttachCommand(argv []string) error {
	cmd := gui.OSCommand.NewCmd(argv[0], argv[1:]...)

	return gui.runSubprocessWithMessage(cmd, gui.Tr.DetachFromContainerShortCut)
}

func (gui *Gui) handleContainerAttachCurrentTerminal(g *gocui.Gui, v *gocui.View) error {
	ctr, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	return gui.attachInCurrentTerminal(ctr)
}

// handleContainerAttachFullLogs is the explicit ctrl+a path: it may replay the
// whole history, so it lives apart from the instant a/A attach.
func (gui *Gui) handleContainerAttachShellTab(g *gocui.Gui, v *gocui.View) error {
	ctr, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	return gui.startContainerTerminal(ctr)
}

func (gui *Gui) handleContainerAttachFullLogs(g *gocui.Gui, v *gocui.View) error {
	ctr, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	if err := ctr.CanAttach(); err != nil {
		return gui.createErrorPanel(err.Error())
	}

	argv := commands.AttachFullLogsTerminalArgv(gui.Config.PodmanBinary(), ctr.ID)
	if err := gui.OSCommand.OpenExternalTerminal(argv); err != nil {
		if errors.Is(err, commands.ErrNoTerminalEmulator) {
			return gui.runAttachCommand(argv)
		}
		return gui.createErrorPanel(err.Error())
	}

	gui.flashStatus(gui.Tr.AttachOpenedInTerminal)

	return nil
}

func (gui *Gui) handlePruneContainers() error {
	return gui.createConfirmationPanel(gui.Tr.Confirm, gui.Tr.ConfirmPruneContainers, func(g *gocui.Gui, v *gocui.View) error {
		return gui.WithWaitingStatus(gui.Tr.PruningStatus, func() error {
			err := gui.PodmanCommand.PruneContainers()
			if err != nil {
				return gui.createErrorPanel(err.Error())
			}
			return nil
		})
	}, nil)
}

func (gui *Gui) handleContainerViewLogs(g *gocui.Gui, v *gocui.View) error {
	ctr, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	gui.renderLogsToStdout(ctr)

	return nil
}

// handleContainersTerminal opens the shell for the selected container, or closes
// the one that is already open, so ctrl+x works as a single toggle.
func (gui *Gui) handleContainersTerminal(g *gocui.Gui, v *gocui.View) error {
	if gui.currentTerminal() != nil {
		return gui.closeContainerTerminal()
	}

	ctr, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	return gui.startContainerTerminal(ctr)
}

func (gui *Gui) handleContainersCustomCommand(g *gocui.Gui, v *gocui.View) error {
	ctr, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	commandObject := gui.PodmanCommand.NewCommandObject(commands.CommandObject{
		Container: ctr,
	})

	customCommands := gui.Config.UserConfig.CustomCommands.Containers

	return gui.createCustomCommandMenu(customCommands, commandObject)
}

func (gui *Gui) handleStopContainers() error {
	return gui.createConfirmationPanel(gui.Tr.Confirm, gui.Tr.ConfirmStopContainers, func(g *gocui.Gui, v *gocui.View) error {
		return gui.WithWaitingStatus(gui.Tr.StoppingStatus, func() error {
			for _, ctr := range gui.Panels.Containers.List.GetAllItems() {
				if err := ctr.Stop(); err != nil {
					gui.Log.Error(err)
				}
			}

			return nil
		})
	}, nil)
}

func (gui *Gui) handleRemoveContainers() error {
	return gui.createConfirmationPanel(gui.Tr.Confirm, gui.Tr.ConfirmRemoveContainers, func(g *gocui.Gui, v *gocui.View) error {
		return gui.WithWaitingStatus(gui.Tr.RemovingStatus, func() error {
			for _, ctr := range gui.Panels.Containers.List.GetAllItems() {
				if err := ctr.Remove(container.RemoveOptions{Force: true}); err != nil {
					gui.Log.Error(err)
				}
			}

			return nil
		})
	}, nil)
}

func (gui *Gui) handleContainersBulkCommand(g *gocui.Gui, v *gocui.View) error {
	baseBulkCommands := []config.CustomCommand{
		{
			Name:             gui.Tr.StopAllContainers,
			InternalFunction: gui.handleStopContainers,
		},
		{
			Name:             gui.Tr.RemoveAllContainers,
			InternalFunction: gui.handleRemoveContainers,
		},
		{
			Name:             gui.Tr.PruneContainers,
			InternalFunction: gui.handlePruneContainers,
		},
		{
			Name:             gui.Tr.AutoUpdateContainers,
			InternalFunction: gui.handleAutoUpdate,
		},
	}

	bulkCommands := append(baseBulkCommands, gui.Config.UserConfig.BulkCommands.Containers...)
	commandObject := gui.PodmanCommand.NewCommandObject(commands.CommandObject{})

	return gui.createBulkCommandMenu(bulkCommands, commandObject)
}

// Open first port in browser
func (gui *Gui) handleContainersOpenInBrowserCommand(g *gocui.Gui, v *gocui.View) error {
	ctr, err := gui.Panels.Containers.GetSelectedItem()
	if err != nil {
		return nil
	}

	return gui.openContainerInBrowser(ctr)
}

func (gui *Gui) openContainerInBrowser(ctr *commands.Container) error {
	// skip if no any ports
	if len(ctr.Container.Ports) == 0 {
		return nil
	}
	// skip if the first port is not published
	port := ctr.Container.Ports[0]
	if port.IP == "" {
		return nil
	}
	ip := port.IP
	if ip == "0.0.0.0" {
		ip = "localhost"
	}
	link := fmt.Sprintf("http://%s:%d/", ip, port.PublicPort)
	return gui.OSCommand.OpenLink(link)
}
