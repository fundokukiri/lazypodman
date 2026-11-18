package gui

import (
	"sync"
	"time"

	"github.com/fundokukiri/lazypodman/pkg/utils"
	"github.com/jesseduffield/gocui"
)

// flashStatusDuration is how long a one-shot result stays on the status line.
const flashStatusDuration = 10 * time.Second

type appStatus struct {
	name       string
	statusType string
	duration   int
}

type statusManager struct {
	mu       sync.Mutex
	statuses []appStatus

	// flash is a one-shot message (a healthcheck result, for instance). It is
	// shown until flashUntil, and a waiting status takes precedence while set.
	flash      string
	flashUntil time.Time

	// now is replaceable in tests.
	now func() time.Time
}

func (m *statusManager) nowTime() time.Time {
	if m.now != nil {
		return m.now()
	}

	return time.Now()
}

func (m *statusManager) removeStatus(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	newStatuses := []appStatus{}
	for _, status := range m.statuses {
		if status.name != name {
			newStatuses = append(newStatuses, status)
		}
	}
	m.statuses = newStatuses
}

func (m *statusManager) addWaitingStatus(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.removeStatusUnlocked(name)
	newStatus := appStatus{
		name:       name,
		statusType: "waiting",
		duration:   0,
	}
	m.statuses = append([]appStatus{newStatus}, m.statuses...)
}

func (m *statusManager) removeStatusUnlocked(name string) {
	newStatuses := []appStatus{}
	for _, status := range m.statuses {
		if status.name != name {
			newStatuses = append(newStatuses, status)
		}
	}
	m.statuses = newStatuses
}

// setFlash records a one-shot message valid for ttl.
func (m *statusManager) setFlash(name string, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.flash = name
	m.flashUntil = m.nowTime().Add(ttl)
}

func (m *statusManager) getStatusString() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.statuses) != 0 {
		topStatus := m.statuses[0]
		if topStatus.statusType == "waiting" {
			return topStatus.name + " " + utils.Loader()
		}

		return topStatus.name
	}

	if m.flash != "" && m.nowTime().Before(m.flashUntil) {
		return m.flash
	}

	return ""
}

// flashStatus shows a message on the status line for a few seconds without
// blocking the caller or the UI. Used for healthcheck results, which are not
// errors and must not open a popup.
func (gui *Gui) flashStatus(message string) {
	gui.statusManager.setFlash(message, flashStatusDuration)
	gui.renderAppStatus()

	time.AfterFunc(flashStatusDuration, func() {
		gui.g.Update(func(g *gocui.Gui) error {
			return gui.renderString(g, "appStatus", gui.statusManager.getStatusString())
		})
	})
}

func (gui *Gui) renderAppStatus() {
	appStatus := gui.statusManager.getStatusString()

	gui.g.Update(func(g *gocui.Gui) error {
		return gui.renderString(g, "appStatus", appStatus)
	})
}

// WithWaitingStatus wraps a function and shows a waiting status while the function is still executing
func (gui *Gui) WithWaitingStatus(name string, f func() error) error {
	go func() {
		gui.statusManager.addWaitingStatus(name)

		defer func() {
			gui.statusManager.removeStatus(name)
			gui.renderAppStatus()
		}()

		go func() {
			ticker := time.NewTicker(time.Millisecond * 50)
			defer ticker.Stop()
			for range ticker.C {
				appStatus := gui.statusManager.getStatusString()
				if appStatus == "" {
					return
				}
				if err := gui.renderString(gui.g, "appStatus", appStatus); err != nil {
					gui.Log.Warn(err)
				}
			}
		}()

		if err := f(); err != nil {
			gui.g.Update(func(g *gocui.Gui) error {
				return gui.createErrorPanel(err.Error())
			})
		}
	}()

	return nil
}
