// Package dataplane provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen/v2 version v2.0.0 DO NOT EDIT.
package dataplane

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/oapi-codegen/runtime"
)

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Returns a list of instances.
	// (GET /instances)
	GetInstances(c *gin.Context)
	// Update an instance's configuration.
	// (PUT /instances/{instanceId}/configurations)
	UpdateInstanceConfiguration(c *gin.Context, instanceId string)
	// Returns the latest status of the instance's configuration
	// (GET /instances/{instanceId}/configurations/status)
	GetInstanceConfigurationStatus(c *gin.Context, instanceId string)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandler       func(*gin.Context, error, int)
}

type MiddlewareFunc func(c *gin.Context)

// GetInstances operation middleware
func (siw *ServerInterfaceWrapper) GetInstances(c *gin.Context) {

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.GetInstances(c)
}

// UpdateInstanceConfiguration operation middleware
func (siw *ServerInterfaceWrapper) UpdateInstanceConfiguration(c *gin.Context) {

	var err error

	// ------------- Path parameter "instanceId" -------------
	var instanceId string

	err = runtime.BindStyledParameter("simple", false, "instanceId", c.Param("instanceId"), &instanceId)
	if err != nil {
		siw.ErrorHandler(c, fmt.Errorf("Invalid format for parameter instanceId: %w", err), http.StatusBadRequest)
		return
	}

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.UpdateInstanceConfiguration(c, instanceId)
}

// GetInstanceConfigurationStatus operation middleware
func (siw *ServerInterfaceWrapper) GetInstanceConfigurationStatus(c *gin.Context) {

	var err error

	// ------------- Path parameter "instanceId" -------------
	var instanceId string

	err = runtime.BindStyledParameter("simple", false, "instanceId", c.Param("instanceId"), &instanceId)
	if err != nil {
		siw.ErrorHandler(c, fmt.Errorf("Invalid format for parameter instanceId: %w", err), http.StatusBadRequest)
		return
	}

	for _, middleware := range siw.HandlerMiddlewares {
		middleware(c)
		if c.IsAborted() {
			return
		}
	}

	siw.Handler.GetInstanceConfigurationStatus(c, instanceId)
}

// GinServerOptions provides options for the Gin server.
type GinServerOptions struct {
	BaseURL      string
	Middlewares  []MiddlewareFunc
	ErrorHandler func(*gin.Context, error, int)
}

// RegisterHandlers creates http.Handler with routing matching OpenAPI spec.
func RegisterHandlers(router gin.IRouter, si ServerInterface) {
	RegisterHandlersWithOptions(router, si, GinServerOptions{})
}

// RegisterHandlersWithOptions creates http.Handler with additional options
func RegisterHandlersWithOptions(router gin.IRouter, si ServerInterface, options GinServerOptions) {
	errorHandler := options.ErrorHandler
	if errorHandler == nil {
		errorHandler = func(c *gin.Context, err error, statusCode int) {
			c.JSON(statusCode, gin.H{"msg": err.Error()})
		}
	}

	wrapper := ServerInterfaceWrapper{
		Handler:            si,
		HandlerMiddlewares: options.Middlewares,
		ErrorHandler:       errorHandler,
	}

	router.GET(options.BaseURL+"/instances", wrapper.GetInstances)
	router.PUT(options.BaseURL+"/instances/:instanceId/configurations", wrapper.UpdateInstanceConfiguration)
	router.GET(options.BaseURL+"/instances/:instanceId/configurations/status", wrapper.GetInstanceConfigurationStatus)
}
