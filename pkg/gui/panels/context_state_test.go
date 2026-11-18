package panels

import "testing"

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
