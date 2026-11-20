package gui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/config"
	"github.com/fundokukiri/lazypodman/pkg/gui/panels"
	"github.com/fundokukiri/lazypodman/pkg/gui/presentation"
	"github.com/fundokukiri/lazypodman/pkg/gui/types"
	"github.com/fundokukiri/lazypodman/pkg/tasks"
	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/jesseduffield/gocui"
)

func (gui *Gui) getPodsPanel() *panels.SideListPanel[*commands.Pod] {
	return &panels.SideListPanel[*commands.Pod]{
		ContextState: &panels.ContextState[*commands.Pod]{
			GetMainTabs: func() []panels.MainTab[*commands.Pod] {
				return []panels.MainTab[*commands.Pod]{
					{Key: "stats", Title: gui.Tr.PodStatsTitle, Render: gui.renderPodStats},
					{Key: "logs", Title: gui.Tr.LogsTitle, Render: gui.renderPodLogs},
					{Key: "containers", Title: gui.Tr.PodContainersTitle, Render: gui.renderPodContainers},
					{Key: "config", Title: gui.Tr.ConfigTitle, Render: gui.renderPodConfig},
				}
			},
			GetItemContextCacheKey: func(pod *commands.Pod) string {
				// The status is part of the key so a pod that starts or stops gets
				// fresh content instead of the previous run's logs.
				return "pods-" + pod.ID + "-" + pod.Status
			},
		},
		ListPanel: panels.ListPanel[*commands.Pod]{
			List: panels.NewFilteredList[*commands.Pod](),
			View: gui.Views.Pods,
		},
		NoItemsMessage: gui.Tr.NoPods,
		Gui:            gui.intoInterface(),
		Sort:           sortPods,
		GetTableCells: func(pod *commands.Pod) []string {
			return presentation.GetPodDisplayStrings(pod)
		},
		// The panel is created even when pods are unavailable so the layout and
		// the key table never see a nil panel; it stays hidden instead.
		Hide: func() bool { return !gui.PodsEnabled() },
	}
}

var podStateOrder = map[string]int{
	"running":  1,
	"degraded": 2,
	"paused":   3,
	"created":  4,
	"exited":   5,
	"stopped":  6,
}

func sortPods(a, b *commands.Pod) bool {
	left, leftKnown := podStateOrder[strings.ToLower(a.Status)]
	right, rightKnown := podStateOrder[strings.ToLower(b.Status)]

	if !leftKnown {
		left = len(podStateOrder) + 1
	}

	if !rightKnown {
		right = len(podStateOrder) + 1
	}

	if left == right {
		return a.Name < b.Name
	}

	return left < right
}

// containersForPod returns the socket containers that belong to the pod, using the
// last pod snapshot. The slice is a copy: the caller may keep it while a refresh
// replaces the panel items.
func (gui *Gui) containersForPod(pod *commands.Pod) []*commands.Container {
	state := gui.PodmanCommand.PodStateSnapshot()
	if state == nil {
		return nil
	}

	containers := make([]*commands.Container, 0, pod.NumberOfContainers)
	for _, container := range gui.Panels.Containers.List.GetAllItems() {
		if state.PodByContainer[container.ID] == pod.Name {
			containers = append(containers, container)
		}
	}

	return containers
}

// refreshPods reloads the pods and the container->pod map. It is called from the
// same refresh cycle as the containers, never from the UI goroutine.
// fetchPodState runs the CLI work for pods. It is separate from applyPodState so
// the startup refresh can start it before the container list exists.
func (gui *Gui) fetchPodState() (*commands.PodState, error) {
	if !gui.PodsEnabled() || gui.Views.Pods == nil {
		return nil, nil
	}

	return gui.PodmanCommand.RefreshPodState(context.Background())
}

func (gui *Gui) applyPodState(state *commands.PodState) error {
	gui.Panels.Pods.SetItems(state.Pods)
	gui.annotateContainersWithPods(state)

	return gui.Panels.Pods.RerenderList()
}

func (gui *Gui) refreshPods() error {
	state, err := gui.fetchPodState()
	if err != nil {
		return err
	}

	if state == nil {
		return nil
	}

	return gui.applyPodState(state)
}

func (gui *Gui) renderPodLogs(pod *commands.Pod) tasks.TaskFunc {
	return gui.NewTickerTask(TickerTaskOpts{
		Func: func(ctx context.Context, notifyStopped chan struct{}) {
			gui.renderPodLogsAux(pod, ctx, notifyStopped)
		},
		Duration:   time.Millisecond * 200,
		Before:     func(ctx context.Context) { gui.clearMainView() },
		Wrap:       gui.Config.UserConfig.Gui.WrapMainPanel,
		Autoscroll: true,
	})
}

func (gui *Gui) renderPodLogsAux(pod *commands.Pod, ctx context.Context, notifyStopped chan struct{}) {
	gui.clearMainView()
	defer func() {
		notifyStopped <- struct{}{}
	}()

	containers := gui.containersForPod(pod)

	stream, err := gui.PodmanCommand.Pods().PodLogs(ctx, pod, containers, gui.podLogTail())
	if err != nil {
		gui.Log.Error(err)
		gui.RenderStringMain(err.Error())

		return
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	names := make([]string, 0, len(containers))
	for _, container := range containers {
		names = append(names, container.Name)
	}

	writer := gui.Views.Main
	for scanner.Scan() {
		name, message := commands.ParsePodLogLine(scanner.Text(), names)
		if _, err := fmt.Fprintln(writer, presentation.ColoredPodLogLine(name, message)); err != nil {
			return
		}
	}
}

func (gui *Gui) renderPodStats(pod *commands.Pod) tasks.TaskFunc {
	return gui.NewTickerTask(TickerTaskOpts{
		Func: func(ctx context.Context, notifyStopped chan struct{}) {
			row := commands.AggregatePodStats(gui.containersForPod(pod))
			gui.reRenderStringMain(presentation.RenderPodStats(pod, row))
		},
		Duration:   time.Second,
		Before:     func(ctx context.Context) { gui.clearMainView() },
		Wrap:       false,
		Autoscroll: false,
	})
}

func (gui *Gui) renderPodContainers(pod *commands.Pod) tasks.TaskFunc {
	return gui.NewSimpleRenderStringTask(func() string {
		containers := gui.containersForPod(pod)
		if len(containers) == 0 {
			return gui.Tr.NoPodContainers
		}

		rows := make([][]string, 0, len(containers))
		for _, container := range containers {
			rows = append(rows, []string{
				utils.ColoredString(container.Name, presentation.PodContainerNameColor(container.Name)),
				container.Container.Status,
				container.Container.Image,
			})
		}

		rendered, err := utils.RenderTable(rows)
		if err != nil {
			gui.Log.Error(err)

			return gui.Tr.NoPodContainers
		}

		return rendered
	})
}

// renderPodConfig shows `pod inspect` output. The Docker-compatible API has no
// pod endpoint, so this is the only place the full document is visible.
func (gui *Gui) renderPodConfig(pod *commands.Pod) tasks.TaskFunc {
	return gui.NewSimpleRenderStringTask(func() string {
		client := gui.PodmanCommand.Pods()

		output, err := client.PodInspect(context.Background(), pod.Name)
		if err != nil {
			return err.Error()
		}

		var indented map[string]interface{}
		if err := json.Unmarshal(output, &indented); err != nil {
			return string(output)
		}

		pretty, err := json.MarshalIndent(indented, "", "  ")
		if err != nil {
			return string(output)
		}

		return string(pretty)
	})
}

// podAction runs a lifecycle verb and refreshes everything the pod affects:
// the pod list, and the containers that live in it.
func (gui *Gui) podAction(pod *commands.Pod, verb string, waitingStatus string, extra ...string) error {
	return gui.WithWaitingStatus(waitingStatus, func() error {
		if err := gui.PodmanCommand.Pods().PodAction(context.Background(), verb, pod.Name, extra...); err != nil {
			return gui.createErrorPanel(err.Error())
		}

		if err := gui.refreshContainersAndServices(); err != nil {
			gui.Log.Error(err)
		}

		if err := gui.refreshPods(); err != nil {
			gui.Log.Error(err)
		}

		return nil
	})
}

func (gui *Gui) selectedPod() *commands.Pod {
	pod, err := gui.Panels.Pods.GetSelectedItem()
	if err != nil {
		return nil
	}

	return pod
}

func (gui *Gui) handlePodStart(g *gocui.Gui, v *gocui.View) error {
	pod := gui.selectedPod()
	if pod == nil {
		return nil
	}

	return gui.podAction(pod, "start", gui.Tr.StartingStatus)
}

func (gui *Gui) handlePodStop(g *gocui.Gui, v *gocui.View) error {
	pod := gui.selectedPod()
	if pod == nil {
		return nil
	}

	return gui.createConfirmationPanel(gui.Tr.Confirm, gui.Tr.ConfirmStopPod, func(g *gocui.Gui, v *gocui.View) error {
		return gui.podAction(pod, "stop", gui.Tr.StoppingStatus)
	}, nil)
}

func (gui *Gui) handlePodRestart(g *gocui.Gui, v *gocui.View) error {
	pod := gui.selectedPod()
	if pod == nil {
		return nil
	}

	return gui.podAction(pod, "restart", gui.Tr.RestartingStatus)
}

func (gui *Gui) handlePodPause(g *gocui.Gui, v *gocui.View) error {
	pod := gui.selectedPod()
	if pod == nil {
		return nil
	}

	if strings.EqualFold(pod.Status, "paused") {
		return gui.podAction(pod, "unpause", gui.Tr.PausingStatus)
	}

	return gui.podAction(pod, "pause", gui.Tr.PausingStatus)
}

func (gui *Gui) handlePodRemoveMenu(g *gocui.Gui, v *gocui.View) error {
	pod := gui.selectedPod()
	if pod == nil {
		return nil
	}

	menuItems := []*types.MenuItem{
		{
			LabelColumns: []string{gui.Tr.RemovePod, gui.Config.PodmanBinary() + " pod rm " + pod.Name},
			OnPress: func() error {
				return gui.createConfirmationPanel(gui.Tr.Confirm, gui.Tr.ConfirmRemovePod, func(g *gocui.Gui, v *gocui.View) error {
					return gui.podAction(pod, "rm", gui.Tr.RemovingStatus)
				}, nil)
			},
		},
		{
			LabelColumns: []string{gui.Tr.RemovePodForce, gui.Config.PodmanBinary() + " pod rm --force " + pod.Name},
			OnPress: func() error {
				return gui.createConfirmationPanel(gui.Tr.Confirm, gui.Tr.ConfirmRemovePod, func(g *gocui.Gui, v *gocui.View) error {
					return gui.podAction(pod, "rm", gui.Tr.RemovingStatus, "--force")
				}, nil)
			},
		},
	}

	return gui.Menu(CreateMenuOptions{Title: "", Items: menuItems})
}

func (gui *Gui) handlePodsPrune() error {
	return gui.createConfirmationPanel(gui.Tr.Confirm, gui.Tr.ConfirmPrunePods, func(g *gocui.Gui, v *gocui.View) error {
		return gui.WithWaitingStatus(gui.Tr.PruningStatus, func() error {
			if err := gui.PodmanCommand.Pods().PodPrune(context.Background()); err != nil {
				return gui.createErrorPanel(err.Error())
			}

			return gui.refreshPods()
		})
	}, nil)
}

func (gui *Gui) handlePodsBulkCommand(g *gocui.Gui, v *gocui.View) error {
	bulkCommands := append([]config.CustomCommand{
		{
			Name:             gui.Tr.PrunePods,
			InternalFunction: func() error { return gui.handlePodsPrune() },
		},
	}, gui.Config.UserConfig.BulkCommands.Pods...)

	commandObject := gui.PodmanCommand.NewCommandObject(commands.CommandObject{})

	return gui.createBulkCommandMenu(bulkCommands, commandObject)
}

func (gui *Gui) handlePodsCustomCommand(g *gocui.Gui, v *gocui.View) error {
	pod := gui.selectedPod()
	if pod == nil {
		return nil
	}

	commandObject := gui.PodmanCommand.NewCommandObject(commands.CommandObject{Pod: pod})

	return gui.createCustomCommandMenu(gui.Config.UserConfig.CustomCommands.Pods, commandObject)
}

// podLogTail converts the logs.tail config (a string, because "all" is allowed for
// containers) into the integer `podman logs --tail` needs.
func (gui *Gui) podLogTail() int {
	tail, err := strconv.Atoi(gui.Config.UserConfig.Logs.Tail)
	if err != nil || tail < 0 {
		return 300
	}

	return tail
}

// annotateContainersWithPods stamps the pod name on the containers panel items so
// the containers list can show a pod column without another podman call.
func (gui *Gui) annotateContainersWithPods(state *commands.PodState) {
	if state == nil {
		return
	}

	for _, container := range gui.Panels.Containers.List.GetAllItems() {
		container.PodName = state.PodByContainer[container.ID]
	}

	if err := gui.Panels.Containers.RerenderList(); err != nil {
		gui.Log.Error(err)
	}
}

// podColumnEnabled reports whether the containers panel shows the pod column.
// Both the config option and an enabled pod client are required; without pods
// every cell would be empty, so the column is omitted entirely.
func (gui *Gui) podColumnEnabled() bool {
	return gui.Config.UserConfig.Gui.ShowPodColumn && gui.PodsEnabled()
}
