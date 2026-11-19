package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainerDiskSizeIsUnknownByDefault(t *testing.T) {
	ctr := &Container{}

	rw, rootFs, known := ctr.DiskSize()

	assert.False(t, known)
	assert.Zero(t, rw)
	assert.Zero(t, rootFs)
}

func TestContainerDiskSizeRecordsBothLayers(t *testing.T) {
	ctr := &Container{}
	ctr.SetDiskSize(1234, 987654)

	rw, rootFs, known := ctr.DiskSize()

	assert.True(t, known)
	assert.Equal(t, int64(1234), rw)
	assert.Equal(t, int64(987654), rootFs)
}
