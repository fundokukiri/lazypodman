package gui

import (
	"bytes"
	"strings"
	"time"

	"github.com/fundokukiri/lazypodman/pkg/commands"
)

// logViewCache remembers what the Logs tab showed when the user switched to
// another tab of the same container. Without it every return to Logs replays
// the whole configured tail, so the tab looks like it is loading again.
type logViewCache struct {
	// key is the container id plus its state, the same value the panel uses as
	// its context cache key. A different container or a restarted one misses.
	key     string
	content string
	originY int
	savedAt time.Time
}

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

	gui.logViewCache = &logViewCache{
		key:     gui.logCacheKey(container),
		content: content,
		originY: view.OriginY(),
		savedAt: time.Now(),
	}
}

// restoreLogView puts the cached Logs content back and reports whether it did.
func (gui *Gui) restoreLogView(container *commands.Container, view logViewBuffer) (bool, time.Time) {
	gui.logViewCacheMutex.Lock()
	cache := gui.logViewCache
	gui.logViewCacheMutex.Unlock()

	if cache == nil {
		return false, time.Time{}
	}
	if cache.key != gui.logCacheKey(container) {
		// Visiting another container ends this tab's cache. Coming back should
		// load fresh logs rather than a stale snapshot.
		gui.invalidateLogViewCache()
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
	gui.logViewCache = nil
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
