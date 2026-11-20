package gui

import (
	"testing"

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

func TestLogViewCacheDoesNotSurviveAnotherContainer(t *testing.T) {
	gui := &Gui{}
	first := cachedContainer()
	gui.cacheLogView(first, &fakeLogTarget{content: "old\n"}, gui.logCacheGeneration())

	second := &commands.Container{ID: "other"}
	second.Container.State = "running"
	restored, _ := gui.restoreLogView(second, &fakeLogTarget{})
	assert.False(t, restored)
	restored, _ = gui.restoreLogView(first, &fakeLogTarget{})
	assert.False(t, restored)
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
