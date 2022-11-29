package upstream

import (
	"context"
	"net/http"
	"testing"
	"time"

	conf "github.com/nginx/agent/v2/test/integration/nginx"
	"github.com/stretchr/testify/assert"
)

type Handler struct {
	Handler http.HandlerFunc
}

type HttpTestUpstream struct {
	Name     string
	Address  string
	Handlers map[string]Handler
}

func (u *HttpTestUpstream) AsUpstream() conf.Upstream {
	return conf.Upstream{
		Server: u.Address,
	}
}

type LocationsMarkers = map[string]map[string]string
type LocationsDirectives = map[string][]string

func (u *HttpTestUpstream) AsServer(address string, serverMarkers map[string]string, locationsMarkers LocationsMarkers, locationsDirectives LocationsDirectives) conf.Server {
	return conf.Server{
		Listen:           address,
		F5MetricsMarkers: serverMarkers,
		Locations:        u.locations(locationsMarkers, locationsDirectives),
	}
}

func (u *HttpTestUpstream) locations(locationsMarkers LocationsMarkers, locationsDirectives LocationsDirectives) map[string]conf.Location {
	result := map[string]conf.Location{}
	for name := range u.Handlers {
		result[name] = conf.Location{
			F5MetricsMarkers: locationsMarkers[name],
			UpstreamName:     u.Name,
			Directives:       locationsDirectives[name],
		}
	}
	return result
}

func (u *HttpTestUpstream) Serve(t *testing.T) {
	handler := http.NewServeMux()
	for k, v := range u.Handlers {
		handler.HandleFunc(k, v.Handler)
	}
	server := http.Server{
		Addr:         u.Address,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		err := server.ListenAndServe()
		assert.ErrorIs(t, err, http.ErrServerClosed)
	}()

	t.Cleanup(func() {
		err := server.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}
