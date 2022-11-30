package api

import (
    "context"
    "fmt"
	"net/http"
    "testing"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

type nginxContainer struct {
    testcontainers.Container
    URI string
}

func setupNginx(ctx context.Context) (*nginxContainer, error) {
    req := testcontainers.ContainerRequest{
        Image:        "nginx",
        ExposedPorts: []string{"80/tcp"},
        WaitingFor:   wait.ForHTTP("/"),
    }
    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        return nil, err
    }

    ip, err := container.Host(ctx)
    if err != nil {
        return nil, err
    }

    mappedPort, err := container.MappedPort(ctx, "80")
    if err != nil {
        return nil, err
    }

    uri := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())

    return &nginxContainer{Container: container, URI: uri}, nil
}

func TestIntegrationNginxLatestReturn(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()

    nginxC, err := setupNginx(ctx)
    if err != nil {
        t.Fatal(err)
    }

    // Clean up the container after the test is complete
    t.Cleanup(func() {
        if err := nginxC.Terminate(ctx); err != nil {
            t.Fatalf("failed to terminate container: %s", err)
        }
    })

    resp, err := http.Get(nginxC.URI)
    if err != nil {
        t.Fatal(err)
    }

    if resp.StatusCode != http.StatusOK {
        t.Fatalf("Expected status code %d. Got %d.", http.StatusOK, resp.StatusCode)
    }
}