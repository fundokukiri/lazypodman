package gui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/fundokukiri/lazypodman/pkg/gui/types"
	"github.com/jesseduffield/gocui"
)

type podmanConnection struct {
	Name    string `json:"Name"`
	URI     string `json:"URI"`
	Default bool   `json:"Default"`
}

// handleConnectionsMenu lists `podman system connection ls` and restarts
// lazypodman against the chosen endpoint. Changing the client at runtime would
// leave every existing panel and stream pointing at the old store, so a restart
// is the only correct switch.
func (gui *Gui) handleConnectionsMenu(_ *gocui.Gui, _ *gocui.View) error {
	return gui.WithWaitingStatus(gui.trString(gui.Tr.ConnectionsLoadingStatus, "Loading podman connections"), func() error {
		output, err := gui.runPodmanOutput("system", "connection", "ls", "--format", "json")
		if err != nil {
			return fmt.Errorf("%s", strings.TrimSpace(output+"\n"+err.Error()))
		}

		connections, err := parsePodmanConnections([]byte(output))
		if err != nil {
			return err
		}
		if len(connections) == 0 {
			gui.flashStatus(gui.Tr.ConnectionsNone)

			return nil
		}

		items := make([]*types.MenuItem, 0, len(connections))
		for _, connection := range connections {
			connection := connection
			label := connection.Name
			if connection.Default {
				label += " (default)"
			}
			items = append(items, &types.MenuItem{
				LabelColumns: []string{label, color.New(color.FgYellow).Sprint(connection.URI)},
				OnPress:      func() error { return gui.confirmRestartConnection(connection) },
			})
		}

		gui.g.Update(func(*gocui.Gui) error {
			if err := gui.Menu(CreateMenuOptions{Title: gui.Tr.ConnectionsTitle, Items: items}); err != nil {
				return gui.createErrorPanel(err.Error())
			}

			return nil
		})

		return nil
	})
}

func parsePodmanConnections(data []byte) ([]podmanConnection, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" || trimmed == "[]" {
		return nil, nil
	}

	var connections []podmanConnection
	if err := json.Unmarshal([]byte(trimmed), &connections); err != nil {
		return nil, fmt.Errorf("could not read podman connections: %w", err)
	}

	return connections, nil
}

func (gui *Gui) confirmRestartConnection(connection podmanConnection) error {
	prompt := fmt.Sprintf(gui.Tr.ConnectionsRestartPrompt, connection.Name, connection.URI)

	return gui.createConfirmationPanel(gui.Tr.ConnectionsTitle, prompt, func(g *gocui.Gui, v *gocui.View) error {
		if err := restartWithPodmanHost(connection.URI); err != nil {
			return gui.createErrorPanel(err.Error())
		}

		return nil
	}, nil)
}

// withPodmanHostEnv removes every endpoint variable this project reads and sets
// the chosen one, so an old PODMAN_HOST cannot win over the new connection.
func withPodmanHostEnv(env []string, uri string) []string {
	out := make([]string, 0, len(env)+2)
	for _, entry := range env {
		key := entry
		if index := strings.IndexByte(entry, '='); index >= 0 {
			key = entry[:index]
		}
		if key == "PODMAN_HOST" || key == "CONTAINER_HOST" || key == "DOCKER_HOST" {
			continue
		}
		out = append(out, entry)
	}

	return append(out, "CONTAINER_HOST="+uri, "DOCKER_HOST="+uri)
}
