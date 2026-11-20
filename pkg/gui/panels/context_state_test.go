package panels

import (
	"fmt"
	"testing"
)

func TestGetCurrentMainTabClampsWhenTabsShrink(t *testing.T) {
	state := &ContextState[int]{
		GetMainTabs: func() []MainTab[int] {
			return []MainTab[int]{{Key: "logs"}, {Key: "shell"}}
		},
	}

	state.SetMainTabIndex(1)

	if tab := state.GetCurrentMainTab(); tab.Key != "shell" {
		t.Fatalf("expected shell tab, got %q", tab.Key)
	}

	// The shell tab disappears when the shell is closed.
	state.GetMainTabs = func() []MainTab[int] {
		return []MainTab[int]{{Key: "logs"}}
	}

	if tab := state.GetCurrentMainTab(); tab.Key != "logs" {
		t.Fatalf("expected logs tab after shrink, got %q", tab.Key)
	}
}

func TestSyncItemResetsMainTabWhenTheItemChanges(t *testing.T) {
	state := &ContextState[int]{
		GetMainTabs: func() []MainTab[int] {
			return []MainTab[int]{{Key: "logs"}, {Key: "env"}}
		},
		GetItemContextCacheKey: func(item int) string {
			return fmt.Sprintf("%d", item)
		},
	}

	state.SyncItem(1)
	state.SetMainTabIndex(1)
	state.SyncItem(1)

	if tab := state.GetCurrentMainTab(); tab.Key != "env" {
		t.Fatalf("expected the same item to keep its tab, got %q", tab.Key)
	}

	state.SyncItem(2)

	if tab := state.GetCurrentMainTab(); tab.Key != "logs" {
		t.Fatalf("expected a new item to start on logs, got %q", tab.Key)
	}
}
