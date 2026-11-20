package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultDownloadNameAvoidsHostRoot(t *testing.T) {
	assert.Equal(t, "container-root", defaultDownloadName("/"))
	assert.Equal(t, "container-root", defaultDownloadName("."))
	assert.Equal(t, "container-root", defaultDownloadName(".."))
	assert.Equal(t, "etc", defaultDownloadName("/etc/"))
	assert.Equal(t, "hosts", defaultDownloadName("/etc/hosts"))
}
