// Code generated by counterfeiter. DO NOT EDIT.
package servicefakes

import (
	"sync"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service"
)

type FakeInstanceServiceInterface struct {
	GetInstancesStub        func([]*model.Process) []*instances.Instance
	getInstancesMutex       sync.RWMutex
	getInstancesArgsForCall []struct {
		arg1 []*model.Process
	}
	getInstancesReturns struct {
		result1 []*instances.Instance
	}
	getInstancesReturnsOnCall map[int]struct {
		result1 []*instances.Instance
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeInstanceServiceInterface) GetInstances(arg1 []*model.Process) []*instances.Instance {
	var arg1Copy []*model.Process
	if arg1 != nil {
		arg1Copy = make([]*model.Process, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.getInstancesMutex.Lock()
	ret, specificReturn := fake.getInstancesReturnsOnCall[len(fake.getInstancesArgsForCall)]
	fake.getInstancesArgsForCall = append(fake.getInstancesArgsForCall, struct {
		arg1 []*model.Process
	}{arg1Copy})
	stub := fake.GetInstancesStub
	fakeReturns := fake.getInstancesReturns
	fake.recordInvocation("GetInstances", []interface{}{arg1Copy})
	fake.getInstancesMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeInstanceServiceInterface) GetInstancesCallCount() int {
	fake.getInstancesMutex.RLock()
	defer fake.getInstancesMutex.RUnlock()
	return len(fake.getInstancesArgsForCall)
}

func (fake *FakeInstanceServiceInterface) GetInstancesCalls(stub func([]*model.Process) []*instances.Instance) {
	fake.getInstancesMutex.Lock()
	defer fake.getInstancesMutex.Unlock()
	fake.GetInstancesStub = stub
}

func (fake *FakeInstanceServiceInterface) GetInstancesArgsForCall(i int) []*model.Process {
	fake.getInstancesMutex.RLock()
	defer fake.getInstancesMutex.RUnlock()
	argsForCall := fake.getInstancesArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeInstanceServiceInterface) GetInstancesReturns(result1 []*instances.Instance) {
	fake.getInstancesMutex.Lock()
	defer fake.getInstancesMutex.Unlock()
	fake.GetInstancesStub = nil
	fake.getInstancesReturns = struct {
		result1 []*instances.Instance
	}{result1}
}

func (fake *FakeInstanceServiceInterface) GetInstancesReturnsOnCall(i int, result1 []*instances.Instance) {
	fake.getInstancesMutex.Lock()
	defer fake.getInstancesMutex.Unlock()
	fake.GetInstancesStub = nil
	if fake.getInstancesReturnsOnCall == nil {
		fake.getInstancesReturnsOnCall = make(map[int]struct {
			result1 []*instances.Instance
		})
	}
	fake.getInstancesReturnsOnCall[i] = struct {
		result1 []*instances.Instance
	}{result1}
}

func (fake *FakeInstanceServiceInterface) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.getInstancesMutex.RLock()
	defer fake.getInstancesMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeInstanceServiceInterface) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ service.InstanceServiceInterface = new(FakeInstanceServiceInterface)
