package gui

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/fundokukiri/lazypodman/pkg/commands"
	"github.com/stretchr/testify/assert"
)

func cachedContainer() *commands.Container {
	ctr := &commands.Container{ID: "abc"}
	ctr.Container.State = "running"

	return ctr
}

func TestLogViewCacheRestoresTheSameContainer(t *testing.T) {
	gui := &Gui{}
	ctr := cachedContainer()
	source := &fakeLogTarget{content: "one\ntwo\n", originY: 1}
	gui.cacheLogView(ctr, source, gui.logCacheGeneration())

	target := &fakeLogTarget{content: "stats\n"}
	restored, _ := gui.restoreLogView(ctr, target)
	assert.True(t, restored)
	assert.Equal(t, "one\ntwo\n", target.content)
	assert.Equal(t, 1, target.originY)
}

func TestLogViewCacheMissesAfterStateChange(t *testing.T) {
	gui := &Gui{}
	ctr := cachedContainer()
	source := &fakeLogTarget{content: "one\n"}
	gui.cacheLogView(ctr, source, gui.logCacheGeneration())

	ctr.Container.State = "exited"
	target := &fakeLogTarget{content: "stats\n"}
	restored, _ := gui.restoreLogView(ctr, target)
	assert.False(t, restored)
	assert.Equal(t, "stats\n", target.content)
}

func TestLogViewCacheMissesAfterRestart(t *testing.T) {
	gui := &Gui{}
	ctr := cachedContainer()
	ctr.Details.ContainerJSONBase = &container.ContainerJSONBase{}
	ctr.Details.State = &container.State{StartedAt: "2026-01-01T00:00:00Z"}
	gui.cacheLogView(ctr, &fakeLogTarget{content: "old run\n"}, gui.logCacheGeneration())

	ctr.Details.State.StartedAt = "2026-01-01T00:01:00Z"
	restored, _ := gui.restoreLogView(ctr, &fakeLogTarget{})
	assert.False(t, restored)
}

func TestLogViewCacheCanBeInvalidated(t *testing.T) {
	gui := &Gui{}
	ctr := cachedContainer()
	gui.cacheLogView(ctr, &fakeLogTarget{content: "one\n"}, gui.logCacheGeneration())
	gui.invalidateLogViewCache()

	restored, _ := gui.restoreLogView(ctr, &fakeLogTarget{})
	assert.False(t, restored)
}

func TestLogTrimWriterAdoptsRestoredContent(t *testing.T) {
	target := &fakeLogTarget{height: 1, content: "one\n"}
	writer := newLogTrimWriter(target, nil, nil)
	writer.adoptCurrentContent()

	_, err := writer.Write([]byte("two\n"))
	assert.NoError(t, err)
	assert.Equal(t, "one\ntwo\n", target.content)
	assert.Equal(t, 1, target.originY)
}

func TestLogViewCacheSurvivesAnotherContainer(t *testing.T) {
	// Browsing the list with j/k should not reload the logs of a container you
	// come back to; each container keeps its own entry.
	gui := &Gui{}
	first := cachedContainer()
	gui.cacheLogView(first, &fakeLogTarget{content: "first\n"}, gui.logCacheGeneration())

	second := &commands.Container{ID: "other"}
	second.Container.State = "running"
	gui.cacheLogView(second, &fakeLogTarget{content: "second\n"}, gui.logCacheGeneration())

	target := &fakeLogTarget{content: "stats\n"}
	restored, _ := gui.restoreLogView(first, target)
	assert.True(t, restored)
	assert.Equal(t, "first\n", target.content)
}

func TestLogViewCacheKeepsOnlyTheMostRecentEntries(t *testing.T) {
	gui := &Gui{}
	containers := make([]*commands.Container, 0, maxCachedLogViews+1)

	for i := 0; i <= maxCachedLogViews; i++ {
		ctr := &commands.Container{ID: string(rune('a' + i))}
		ctr.Container.State = "running"
		containers = append(containers, ctr)
		gui.cacheLogView(ctr, &fakeLogTarget{content: ctr.ID + "\n"}, gui.logCacheGeneration())
	}

	// The first one was evicted, the last one is still there.
	restored, _ := gui.restoreLogView(containers[0], &fakeLogTarget{})
	assert.False(t, restored)
	restored, _ = gui.restoreLogView(containers[maxCachedLogViews], &fakeLogTarget{})
	assert.True(t, restored)
}

func TestLogViewCacheDoesNotSaveAfterInvalidation(t *testing.T) {
	gui := &Gui{}
	ctr := cachedContainer()
	generation := gui.logCacheGeneration()
	gui.invalidateLogViewCache()

	gui.cacheLogView(ctr, &fakeLogTarget{content: "old\n"}, generation)
	restored, _ := gui.restoreLogView(ctr, &fakeLogTarget{})
	assert.False(t, restored)
}
