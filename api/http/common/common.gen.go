// Package common provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen/v2 version v2.0.0 DO NOT EDIT.
package common

import (
	"encoding/json"
	"errors"

	"github.com/oapi-codegen/runtime"
)

// Defines values for InstanceType.
const (
	AGENT     InstanceType = "AGENT"
	CUSTOM    InstanceType = "CUSTOM"
	NGF       InstanceType = "NGF"
	NGINX     InstanceType = "NGINX"
	NGINXPLUS InstanceType = "NGINXPLUS"
	NIC       InstanceType = "NIC"
	SYSTEM    InstanceType = "SYSTEM"
	UNIT      InstanceType = "UNIT"
)

// Defines values for MessageType.
const (
	ATTACHMENT       MessageType = "ATTACHMENT"
	CERTIFICATE      MessageType = "CERTIFICATE"
	CONFIG           MessageType = "CONFIG"
	CONFIGRESPONSE   MessageType = "CONFIG_RESPONSE"
	CONTROL          MessageType = "CONTROL"
	CONTROLRESPONSE  MessageType = "CONTROL_RESPONSE"
	DATAPLANEREQUEST MessageType = "DATAPLANE_REQUEST"
	DATAPLANEUPDATE  MessageType = "DATAPLANE_UPDATE"
	FILESRESPONSE    MessageType = "FILES_RESPONSE"
	HEARTBEAT        MessageType = "HEARTBEAT"
)

// Defines values for MetaType.
const (
	AGENTMETA MetaType = "AGENT_META"
	NGINXMETA MetaType = "NGINX_META"
)

// AgentMeta defines model for AgentMeta.
type AgentMeta struct {
	Config       *string `json:"config,omitempty"`
	Metrics      *string `json:"metrics,omitempty"`
	Registration *string `json:"registration,omitempty"`

	// Type The type of metadata
	Type MetaType `json:"type"`
}

// ErrorResponse defines model for ErrorResponse.
type ErrorResponse struct {
	Message string `json:"message"`
}

// Instance defines model for Instance.
type Instance struct {
	InstanceId *string        `json:"instanceId,omitempty"`
	Meta       *Instance_Meta `json:"meta,omitempty"`

	// Type The type of a data plane instance
	Type    *InstanceType `json:"type,omitempty"`
	Version *string       `json:"version,omitempty"`
}

// Instance_Meta defines model for Instance.Meta.
type Instance_Meta struct {
	union json.RawMessage
}

// InstanceType The type of a data plane instance
type InstanceType string

// MessageType The available message types on the Management and Data plane APIs
type MessageType string

// MetaType The type of metadata
type MetaType string

// NginxMeta defines model for NginxMeta.
type NginxMeta struct {
	LoadableModules *string `json:"loadable_modules,omitempty"`
	RunnableModules *string `json:"runnable_modules,omitempty"`

	// Type The type of metadata
	Type MetaType `json:"type"`
}

// InternalServerError defines model for InternalServerError.
type InternalServerError = ErrorResponse

// NotFound defines model for NotFound.
type NotFound = ErrorResponse

// AsNginxMeta returns the union data inside the Instance_Meta as a NginxMeta
func (t Instance_Meta) AsNginxMeta() (NginxMeta, error) {
	var body NginxMeta
	err := json.Unmarshal(t.union, &body)
	return body, err
}

// FromNginxMeta overwrites any union data inside the Instance_Meta as the provided NginxMeta
func (t *Instance_Meta) FromNginxMeta(v NginxMeta) error {
	v.Type = "NginxMeta"
	b, err := json.Marshal(v)
	t.union = b
	return err
}

// MergeNginxMeta performs a merge with any union data inside the Instance_Meta, using the provided NginxMeta
func (t *Instance_Meta) MergeNginxMeta(v NginxMeta) error {
	v.Type = "NginxMeta"
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	merged, err := runtime.JsonMerge(t.union, b)
	t.union = merged
	return err
}

// AsAgentMeta returns the union data inside the Instance_Meta as a AgentMeta
func (t Instance_Meta) AsAgentMeta() (AgentMeta, error) {
	var body AgentMeta
	err := json.Unmarshal(t.union, &body)
	return body, err
}

// FromAgentMeta overwrites any union data inside the Instance_Meta as the provided AgentMeta
func (t *Instance_Meta) FromAgentMeta(v AgentMeta) error {
	v.Type = "AgentMeta"
	b, err := json.Marshal(v)
	t.union = b
	return err
}

// MergeAgentMeta performs a merge with any union data inside the Instance_Meta, using the provided AgentMeta
func (t *Instance_Meta) MergeAgentMeta(v AgentMeta) error {
	v.Type = "AgentMeta"
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	merged, err := runtime.JsonMerge(t.union, b)
	t.union = merged
	return err
}

func (t Instance_Meta) Discriminator() (string, error) {
	var discriminator struct {
		Discriminator string `json:"type"`
	}
	err := json.Unmarshal(t.union, &discriminator)
	return discriminator.Discriminator, err
}

func (t Instance_Meta) ValueByDiscriminator() (interface{}, error) {
	discriminator, err := t.Discriminator()
	if err != nil {
		return nil, err
	}
	switch discriminator {
	case "AgentMeta":
		return t.AsAgentMeta()
	case "NginxMeta":
		return t.AsNginxMeta()
	default:
		return nil, errors.New("unknown discriminator value: " + discriminator)
	}
}

func (t Instance_Meta) MarshalJSON() ([]byte, error) {
	b, err := t.union.MarshalJSON()
	return b, err
}

func (t *Instance_Meta) UnmarshalJSON(b []byte) error {
	err := t.union.UnmarshalJSON(b)
	return err
}
