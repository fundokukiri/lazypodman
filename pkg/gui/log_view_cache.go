package gui

import (
	"bytes"
	"strings"
	"time"

	"github.com/fundokukiri/lazypodman/pkg/commands"
)

// logViewCache remembers what the Logs tab showed for one container. Without it
// every return to Logs replays the whole configured tail, so the tab looks like
// it is loading again. Entries live in gui.logViewCaches, keyed by container and
// state, so switching container and coming back does not reload either.
type logViewCache struct {
	content string
	originY int
	savedAt time.Time
}

// maxCachedLogViews bounds how many containers keep a rendered Logs view. Each
// entry is already capped by the Logs.MaxBytes budget (1 MiB by default).
// shortcut: a count, not a byte budget; accepted because 4 x MaxBytes is small
// and this only ever holds what the view already had in memory.
const maxCachedLogViews = 4

// logViewBuffer is the subset of *gocui.View the cache touches. It keeps the
// cache testable without constructing a full gocui view.
type logViewBuffer interface {
	BufferLines() []string
	SetContent(string)
	OriginY() int
	SetOriginY(int) error
}

func (gui *Gui) logCacheKey(container *commands.Container) string {
	key := container.ID + "-" + container.Container.State
	if details := container.DetailsSnapshot(); details.ContainerJSONBase != nil && details.State != nil && details.State.StartedAt != "" {
		key += "-" + details.State.StartedAt
	}

	return key
}

func (gui *Gui) logCacheGeneration() int {
	gui.logViewCacheMutex.Lock()
	defer gui.logViewCacheMutex.Unlock()

	return gui.logViewCacheGeneration
}

func (gui *Gui) cacheLogView(container *commands.Container, view logViewBuffer, generation int) {
	content := strings.Join(view.BufferLines(), "\n")
	if content == "" {
		return
	}

	gui.logViewCacheMutex.Lock()
	defer gui.logViewCacheMutex.Unlock()

	if generation != gui.logViewCacheGeneration {
		return
	}

	if gui.logViewCaches == nil {
		gui.logViewCaches = map[string]*logViewCache{}
	}

	key := gui.logCacheKey(container)
	gui.logViewCaches[key] = &logViewCache{
		content: content,
		originY: view.OriginY(),
		savedAt: time.Now(),
	}

	// Save order doubles as the eviction order: re-saving a container moves it
	// to the back, so a long list never grows past maxCachedLogViews.
	order := make([]string, 0, len(gui.logViewCacheOrder)+1)

	for _, cachedKey := range gui.logViewCacheOrder {
		if cachedKey != key {
			order = append(order, cachedKey)
		}
	}

	gui.logViewCacheOrder = append(order, key)

	for len(gui.logViewCacheOrder) > maxCachedLogViews {
		delete(gui.logViewCaches, gui.logViewCacheOrder[0])
		gui.logViewCacheOrder = gui.logViewCacheOrder[1:]
	}
}

// restoreLogView puts a cached Logs content back and reports whether it did.
func (gui *Gui) restoreLogView(container *commands.Container, view logViewBuffer) (bool, time.Time) {
	gui.logViewCacheMutex.Lock()
	cache := gui.logViewCaches[gui.logCacheKey(container)]
	gui.logViewCacheMutex.Unlock()

	if cache == nil {
		// First visit of this container, or it restarted: read a fresh tail.
		return false, time.Time{}
	}

	view.SetContent(cache.content)
	_ = view.SetOriginY(cache.originY)

	return true, cache.savedAt
}

func (gui *Gui) invalidateLogViewCache() {
	gui.logViewCacheMutex.Lock()
	defer gui.logViewCacheMutex.Unlock()

	gui.logViewCacheGeneration++
	gui.logViewCaches = nil
	gui.logViewCacheOrder = nil
}

// adoptCurrentContent seeds the writer with what is already in the view, so the
// byte budget, trimming and bottom pinning keep working after a restore.
func (w *logTrimWriter) adoptCurrentContent() {
	if w.view == nil {
		return
	}

	content := strings.Join(w.view.BufferLines(), "\n")
	w.bytes = len(content)
	w.lines = bytes.Count([]byte(content), []byte("\n"))
	if !strings.HasSuffix(content, "\n") && content != "" {
		w.lines++
	}
	w.flushed = true
}
