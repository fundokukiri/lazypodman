package commands

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
)

// podmanContainerListResponse is what Podman's Docker-compatible
// /containers/json returns for a two-service compose project that also has a
// hidden pod infra container in it.
const podmanContainerListResponse = `[
  {
    "Id": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "Names": ["/demo-web-1"],
    "Image": "docker.io/library/nginx:latest",
    "Command": "nginx -g 'daemon off;'",
    "State": "running",
    "Status": "Up 2 minutes",
    "Labels": {
      "com.docker.compose.project": "demo",
      "com.docker.compose.service": "web"
    }
  },
  {
    "Id": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    "Names": ["/demo-db-1"],
    "Image": "docker.io/library/postgres:16",
    "Command": "postgres",
    "State": "exited",
    "Status": "Exited (0) 1 minute ago",
    "Labels": {
      "io.podman.compose.project": "demo",
      "io.podman.compose.service": "db"
    }
  },
  {
    "Id": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
    "Names": ["/demo-pod-infra"],
    "Image": "k8s.gcr.io/pause:3.9",
    "Command": "/pause",
    "State": "running",
    "Status": "Up 2 minutes",
    "Labels": {}
  }
]`

func newFakePodmanCommand(t *testing.T) *PodmanCommand {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/containers/json"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(podmanContainerListResponse))
		case strings.Contains(r.URL.Path, "/containers/") && strings.HasSuffix(r.URL.Path, "/json"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Id":"aaaaaaaa","State":{"Status":"running","Running":true}}`))
		default:
			t.Logf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	cli, err := client.NewClientWithOpts(
		client.WithHost("tcp://"+strings.TrimPrefix(server.URL, "http://")),
		client.WithVersion("1.41"),
	)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	return &PodmanCommand{
		Log:       NewDummyLog(),
		OSCommand: NewDummyOSCommand(),
		Client:    cli,
	}
}

func TestGetContainersHidesInfraContainers(t *testing.T) {
	podmanCommand := newFakePodmanCommand(t)

	containers, err := podmanCommand.GetContainers(nil)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.Len(t, containers, 2) {
		t.FailNow()
	}

	names := []string{containers[0].Name, containers[1].Name}
	assert.Equal(t, []string{"demo-web-1", "demo-db-1"}, names)
}

func TestGetContainersReadsPodmanComposeLabels(t *testing.T) {
	podmanCommand := newFakePodmanCommand(t)

	containers, err := podmanCommand.GetContainers(nil)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Len(t, containers, 2) {
		t.FailNow()
	}

	// Docker-compatible labels
	assert.Equal(t, "demo", containers[0].ProjectName)
	assert.Equal(t, "web", containers[0].ServiceName)

	// podman-compose's own io.podman.compose.* labels
	assert.Equal(t, "demo", containers[1].ProjectName)
	assert.Equal(t, "db", containers[1].ServiceName)

	assert.Equal(t, []string{"demo"}, podmanCommand.GetProjectNames(containers))
}
