package gui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/fatih/color"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/fundokukiri/lazypodman/pkg/i18n"
	"github.com/fundokukiri/lazypodman/pkg/tasks"
	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/jesseduffield/gocui"
)

const (
	// logSearchTail bounds the request per container. Searching every line of a
	// long-running container would be slow and could read gigabytes.
	logSearchTail = 500
	// logSearchMatchesPerContainer keeps one noisy container from hiding the rest.
	logSearchMatchesPerContainer = 3
	// logSearchMaxMatchLine keeps a single TUI frame line from filling the screen.
	logSearchMaxMatchLine = 400
	logSearchReadLimit    = 4 << 20
	logSearchTimeout      = 5 * time.Second
)

type logSearchResult struct {
	Container *commands.Container
	Matches   []string
	Err       error
}

func (gui *Gui) handleLogSearch(g *gocui.Gui, v *gocui.View) error {
	return gui.createPromptPanel(gui.Tr.LogSearchTitle, func(g *gocui.Gui, v *gocui.View) error {
		query := strings.TrimSpace(gui.trimmedContent(v))
		if query == "" {
			gui.flashStatus(gui.Tr.LogSearchEmpty)

			return nil
		}

		gui.State.Panels.Main.ObjectKey = "log-search"

		containers := gui.Panels.Containers.List.GetAllItems()
		if err := gui.QueueTask(gui.logSearchTask(query, containers)); err != nil {
			return err
		}

		return gui.switchFocus(gui.Views.Main)
	})
}

func (gui *Gui) logSearchTask(query string, containers []*commands.Container) tasks.TaskFunc {
	return gui.NewTask(TaskOpts{
		Autoscroll: false,
		Wrap:       false,
		Func: func(ctx context.Context) {
			gui.clearMainView()

			results := make([]logSearchResult, 0, len(containers))
			gui.reRenderStringMain(logSearchReport(gui.Tr, query, results, len(containers)))

			for _, ctr := range containers {
				if ctx.Err() != nil {
					return
				}

				matches, err := gui.searchContainerLogs(ctx, ctr, query)
				results = append(results, logSearchResult{Container: ctr, Matches: matches, Err: err})
				gui.reRenderStringMain(logSearchReport(gui.Tr, query, results, len(containers)))
			}
		},
	})
}

func (gui *Gui) searchContainerLogs(ctx context.Context, ctr *commands.Container, query string) ([]string, error) {
	if gui.PodmanCommand == nil || gui.PodmanCommand.Client == nil {
		return nil, fmt.Errorf("podman client unavailable")
	}

	readCtx, cancel := context.WithTimeout(ctx, logSearchTimeout)
	defer cancel()

	readCloser, err := gui.PodmanCommand.Client.ContainerLogs(readCtx, ctr.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", logSearchTail),
		Follow:     false,
	})
	if err != nil {
		return nil, err
	}
	defer readCloser.Close()

	var buf bytes.Buffer
	limited := io.LimitReader(readCloser, logSearchReadLimit)

	if ctr.DetailsLoaded() && ctr.DetailsSnapshot().Config != nil && ctr.DetailsSnapshot().Config.Tty {
		_, err = io.Copy(&buf, limited)
	} else {
		_, err = stdcopy.StdCopy(&buf, &buf, limited)
	}
	if err != nil && buf.Len() == 0 {
		return nil, err
	}

	return findLogMatches(strings.Split(buf.String(), "\n"), query, logSearchMatchesPerContainer), nil
}

// findLogMatches returns at most limit readable lines containing query. The match
// is case-insensitive and ignores terminal colour codes.
func findLogMatches(lines []string, query string, limit int) []string {
	needle := strings.ToLower(query)
	matches := make([]string, 0, limit)

	for _, raw := range lines {
		if len(matches) == limit {
			break
		}

		line := strings.TrimRight(utils.Decolorise(raw), "\r\n\t ")
		if line == "" || !strings.Contains(strings.ToLower(line), needle) {
			continue
		}
		if len(line) > logSearchMaxMatchLine {
			line = line[:logSearchMaxMatchLine] + "…"
		}
		matches = append(matches, line)
	}

	return matches
}

func logSearchReport(tr *i18n.TranslationSet, query string, results []logSearchResult, totalContainers int) string {
	var b strings.Builder

	fmt.Fprintf(&b, tr.LogSearchSummary+"\n\n", query, logSearchTail, totalContainers)

	matches := 0
	for _, result := range results {
		name := result.Container.Name
		if name == "" {
			name = result.Container.ID
		}
		if result.Err != nil {
			fmt.Fprintf(&b, "%s: %s: %v\n", name, tr.LogSearchError, result.Err)
			continue
		}
		if len(result.Matches) == 0 {
			continue
		}

		matches += len(result.Matches)
		fmt.Fprintf(&b, "%s %s\n", utils.ColoredString(name, color.FgYellow), utils.ColoredString(fmt.Sprintf("(%d)", len(result.Matches)), color.FgGreen))
		for _, line := range result.Matches {
			fmt.Fprintf(&b, "  %s\n", line)
		}
		b.WriteString("\n")
	}

	if matches == 0 {
		b.WriteString(tr.LogSearchNoMatches)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(tr.LogSearchHint)

	return b.String()
}
