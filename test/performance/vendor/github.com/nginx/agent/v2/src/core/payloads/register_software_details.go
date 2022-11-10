package payloads

import (
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
)

// RegisterWithDataplaneSoftwareDetailsPayload is an internal payload meant to be used as
// part of registration when there are plugins reporting software details.
type RegisterWithDataplaneSoftwareDetailsPayload struct {
	dataplaneSoftwareDetails      map[string]*proto.DataplaneSoftwareDetails
	dataplaneSoftwareDetailsMutex sync.Mutex
}

// NewRegisterWithDataplaneSoftwareDetailsPayload returns a pointer to an instance of a
// RegisterWithDataplaneSoftwareDetailsPayload object.
func NewRegisterWithDataplaneSoftwareDetailsPayload(details map[string]*proto.DataplaneSoftwareDetails) *RegisterWithDataplaneSoftwareDetailsPayload {
	return &RegisterWithDataplaneSoftwareDetailsPayload{
		dataplaneSoftwareDetails: details,
	}
}

// AddDataplaneSoftwareDetails adds the dataplane software details passed into the function to
// the dataplane software details map object that has been sent as part of the payload.
func (p *RegisterWithDataplaneSoftwareDetailsPayload) AddDataplaneSoftwareDetails(pluginName string, details *proto.DataplaneSoftwareDetails) {
	p.dataplaneSoftwareDetailsMutex.Lock()
	p.dataplaneSoftwareDetails[pluginName] = details
	p.dataplaneSoftwareDetailsMutex.Unlock()
}
