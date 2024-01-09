package http

import (
	"context"
	"log"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nginx/agent/v3/internal/apis/http/common"
	"github.com/nginx/agent/v3/internal/apis/http/dataplane"
	"github.com/nginx/agent/v3/internal/models/instances"
	"github.com/nginx/agent/v3/internal/services"
)

const (
	errNotImplemented = "endpoint not implemented yet"
	genericFailureMsg = "request could not be completed"
)

type (
	// Temporary error response.
	ErrorResponse struct {
		Code    int    `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}

	DataplaneServer struct {
		address string
		logger  *slog.Logger
	}
)

func NewDataplaneServer(address string, l *slog.Logger) *DataplaneServer {
	return &DataplaneServer{
		address: address,
		logger:  l,
	}
}

func (dps *DataplaneServer) Run(ctx context.Context) {
	e := echo.New()
	e.Use(middleware.Logger())

	dataplane.RegisterHandlers(e, dps)

	dps.logger.Info("starting server", "address", dps.address)
	err := e.Start(dps.address)
	if err != nil {
		log.Fatalf("startup of dataplane server failed: %s", err.Error())
	}
}

// GET /instances
func (dps *DataplaneServer) GetInstances(ctx echo.Context) error {
	dps.logger.Debug("GetInstances request")

	instanceService := services.NewInstanceService()
	instances, err := instanceService.GetInstances()

	response := []common.Instance{}

	for _, instance := range instances {
		response = append(response, common.Instance{
			InstanceId: &instance.InstanceId,
			Type:       toPtr(mapTypeEnums(instance.Type.String())),
		})
	}

	if err != nil {
		return ctx.JSON(http.StatusNotImplemented, ErrorResponse{
			http.StatusNotImplemented,
			errNotImplemented,
		})
	}

	return ctx.JSON(http.StatusOK, response)
}

func mapTypeEnums(typeString string) common.InstanceType {
	if typeString == instances.Type_NGINX.String() {
		return common.InstanceTypeNGINX
	}
	return common.InstanceTypeCUSTOM
}

func toPtr[T any](value T) *T {
	return &value
}
