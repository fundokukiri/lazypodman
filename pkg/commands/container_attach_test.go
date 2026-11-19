package commands

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/fundokukiri/lazypodman/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

func attachTestContainer() *Container {
	return &Container{
		Tr: i18n.NewTranslationSet(NewDummyLog(), "en"),
		Container: container.Summary{
			State: "running",
		},
	}
}

func TestCanAttachRequiresInspectDetails(t *testing.T) {
	ctr := attachTestContainer()

	assert.Equal(t, ctr.Tr.WaitingForContainerInfo, ctr.CanAttach().Error())
}

func TestCanAttachRequiresOpenStdin(t *testing.T) {
	ctr := attachTestContainer()
	ctr.Details = container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{},
		Config:            &container.Config{OpenStdin: false},
	}

	assert.Equal(t, ctr.Tr.UnattachableContainerError, ctr.CanAttach().Error())
}

func TestCanAttachRejectsStoppedContainer(t *testing.T) {
	ctr := attachTestContainer()
	ctr.Container.State = "exited"
	ctr.Details = container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{},
		Config:            &container.Config{OpenStdin: true},
	}

	assert.Equal(t, ctr.Tr.CannotAttachStoppedContainerError, ctr.CanAttach().Error())
}

func TestCanAttachAcceptsRunningContainerWithStdin(t *testing.T) {
	ctr := attachTestContainer()
	ctr.Details = container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{},
		Config:            &container.Config{OpenStdin: true},
	}

	assert.NoError(t, ctr.CanAttach())
}
