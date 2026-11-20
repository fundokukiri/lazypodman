package commands

import (
	"sync"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
)

func TestContainerDetailsSnapshotSurvivesConcurrentRefresh(t *testing.T) {
	ctr := &Container{}
	first := container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{},
	}
	first.State = &container.State{Running: true}
	second := first
	second.State = &container.State{Running: false}

	ctr.SetDetails(first)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				ctr.SetDetails(second)
				ctr.SetDetails(first)
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				details := ctr.DetailsSnapshot()
				if details.State != nil {
					_ = details.State.Running
				}
			}
		}()
	}
	wg.Wait()

	assert.True(t, ctr.DetailsLoaded())
}
