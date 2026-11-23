package i18n

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Both removal-lock prompts are formatted with the character count and the name,
// so a missing verb would show "%!d(MISSING)" in front of the user.
func TestRemovalLockPromptsFormat(t *testing.T) {
	set := englishSet()

	for _, prompt := range []string{set.RemovalLockPrompt, set.RemovalUnlockPrompt} {
		assert.NotContains(t, fmt.Sprintf(prompt, 4, "web"), "%!")
	}
}
