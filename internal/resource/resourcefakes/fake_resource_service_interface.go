// Code generated by counterfeiter. DO NOT EDIT.
package resourcefakes

import (
	"sync"

	v1 "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
)

type FakeResourceServiceInterface struct {
	AddInstanceStub        func(*bus.Message) (*v1.Resource, error)
	addInstanceMutex       sync.RWMutex
	addInstanceArgsForCall []struct {
		arg1 *bus.Message
	}
	addInstanceReturns struct {
		result1 *v1.Resource
		result2 error
	}
	addInstanceReturnsOnCall map[int]struct {
		result1 *v1.Resource
		result2 error
	}
	DeleteInstanceStub        func(*bus.Message) (*v1.Resource, error)
	deleteInstanceMutex       sync.RWMutex
	deleteInstanceArgsForCall []struct {
		arg1 *bus.Message
	}
	deleteInstanceReturns struct {
		result1 *v1.Resource
		result2 error
	}
	deleteInstanceReturnsOnCall map[int]struct {
		result1 *v1.Resource
		result2 error
	}
	UpdateInstanceStub        func(*bus.Message) (*v1.Resource, error)
	updateInstanceMutex       sync.RWMutex
	updateInstanceArgsForCall []struct {
		arg1 *bus.Message
	}
	updateInstanceReturns struct {
		result1 *v1.Resource
		result2 error
	}
	updateInstanceReturnsOnCall map[int]struct {
		result1 *v1.Resource
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeResourceServiceInterface) AddInstance(arg1 *bus.Message) (*v1.Resource, error) {
	fake.addInstanceMutex.Lock()
	ret, specificReturn := fake.addInstanceReturnsOnCall[len(fake.addInstanceArgsForCall)]
	fake.addInstanceArgsForCall = append(fake.addInstanceArgsForCall, struct {
		arg1 *bus.Message
	}{arg1})
	stub := fake.AddInstanceStub
	fakeReturns := fake.addInstanceReturns
	fake.recordInvocation("AddInstance", []interface{}{arg1})
	fake.addInstanceMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeResourceServiceInterface) AddInstanceCallCount() int {
	fake.addInstanceMutex.RLock()
	defer fake.addInstanceMutex.RUnlock()
	return len(fake.addInstanceArgsForCall)
}

func (fake *FakeResourceServiceInterface) AddInstanceCalls(stub func(*bus.Message) (*v1.Resource, error)) {
	fake.addInstanceMutex.Lock()
	defer fake.addInstanceMutex.Unlock()
	fake.AddInstanceStub = stub
}

func (fake *FakeResourceServiceInterface) AddInstanceArgsForCall(i int) *bus.Message {
	fake.addInstanceMutex.RLock()
	defer fake.addInstanceMutex.RUnlock()
	argsForCall := fake.addInstanceArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeResourceServiceInterface) AddInstanceReturns(result1 *v1.Resource, result2 error) {
	fake.addInstanceMutex.Lock()
	defer fake.addInstanceMutex.Unlock()
	fake.AddInstanceStub = nil
	fake.addInstanceReturns = struct {
		result1 *v1.Resource
		result2 error
	}{result1, result2}
}

func (fake *FakeResourceServiceInterface) AddInstanceReturnsOnCall(i int, result1 *v1.Resource, result2 error) {
	fake.addInstanceMutex.Lock()
	defer fake.addInstanceMutex.Unlock()
	fake.AddInstanceStub = nil
	if fake.addInstanceReturnsOnCall == nil {
		fake.addInstanceReturnsOnCall = make(map[int]struct {
			result1 *v1.Resource
			result2 error
		})
	}
	fake.addInstanceReturnsOnCall[i] = struct {
		result1 *v1.Resource
		result2 error
	}{result1, result2}
}

func (fake *FakeResourceServiceInterface) DeleteInstance(arg1 *bus.Message) (*v1.Resource, error) {
	fake.deleteInstanceMutex.Lock()
	ret, specificReturn := fake.deleteInstanceReturnsOnCall[len(fake.deleteInstanceArgsForCall)]
	fake.deleteInstanceArgsForCall = append(fake.deleteInstanceArgsForCall, struct {
		arg1 *bus.Message
	}{arg1})
	stub := fake.DeleteInstanceStub
	fakeReturns := fake.deleteInstanceReturns
	fake.recordInvocation("DeleteInstance", []interface{}{arg1})
	fake.deleteInstanceMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeResourceServiceInterface) DeleteInstanceCallCount() int {
	fake.deleteInstanceMutex.RLock()
	defer fake.deleteInstanceMutex.RUnlock()
	return len(fake.deleteInstanceArgsForCall)
}

func (fake *FakeResourceServiceInterface) DeleteInstanceCalls(stub func(*bus.Message) (*v1.Resource, error)) {
	fake.deleteInstanceMutex.Lock()
	defer fake.deleteInstanceMutex.Unlock()
	fake.DeleteInstanceStub = stub
}

func (fake *FakeResourceServiceInterface) DeleteInstanceArgsForCall(i int) *bus.Message {
	fake.deleteInstanceMutex.RLock()
	defer fake.deleteInstanceMutex.RUnlock()
	argsForCall := fake.deleteInstanceArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeResourceServiceInterface) DeleteInstanceReturns(result1 *v1.Resource, result2 error) {
	fake.deleteInstanceMutex.Lock()
	defer fake.deleteInstanceMutex.Unlock()
	fake.DeleteInstanceStub = nil
	fake.deleteInstanceReturns = struct {
		result1 *v1.Resource
		result2 error
	}{result1, result2}
}

func (fake *FakeResourceServiceInterface) DeleteInstanceReturnsOnCall(i int, result1 *v1.Resource, result2 error) {
	fake.deleteInstanceMutex.Lock()
	defer fake.deleteInstanceMutex.Unlock()
	fake.DeleteInstanceStub = nil
	if fake.deleteInstanceReturnsOnCall == nil {
		fake.deleteInstanceReturnsOnCall = make(map[int]struct {
			result1 *v1.Resource
			result2 error
		})
	}
	fake.deleteInstanceReturnsOnCall[i] = struct {
		result1 *v1.Resource
		result2 error
	}{result1, result2}
}

func (fake *FakeResourceServiceInterface) UpdateInstance(arg1 *bus.Message) (*v1.Resource, error) {
	fake.updateInstanceMutex.Lock()
	ret, specificReturn := fake.updateInstanceReturnsOnCall[len(fake.updateInstanceArgsForCall)]
	fake.updateInstanceArgsForCall = append(fake.updateInstanceArgsForCall, struct {
		arg1 *bus.Message
	}{arg1})
	stub := fake.UpdateInstanceStub
	fakeReturns := fake.updateInstanceReturns
	fake.recordInvocation("UpdateInstance", []interface{}{arg1})
	fake.updateInstanceMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeResourceServiceInterface) UpdateInstanceCallCount() int {
	fake.updateInstanceMutex.RLock()
	defer fake.updateInstanceMutex.RUnlock()
	return len(fake.updateInstanceArgsForCall)
}

func (fake *FakeResourceServiceInterface) UpdateInstanceCalls(stub func(*bus.Message) (*v1.Resource, error)) {
	fake.updateInstanceMutex.Lock()
	defer fake.updateInstanceMutex.Unlock()
	fake.UpdateInstanceStub = stub
}

func (fake *FakeResourceServiceInterface) UpdateInstanceArgsForCall(i int) *bus.Message {
	fake.updateInstanceMutex.RLock()
	defer fake.updateInstanceMutex.RUnlock()
	argsForCall := fake.updateInstanceArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeResourceServiceInterface) UpdateInstanceReturns(result1 *v1.Resource, result2 error) {
	fake.updateInstanceMutex.Lock()
	defer fake.updateInstanceMutex.Unlock()
	fake.UpdateInstanceStub = nil
	fake.updateInstanceReturns = struct {
		result1 *v1.Resource
		result2 error
	}{result1, result2}
}

func (fake *FakeResourceServiceInterface) UpdateInstanceReturnsOnCall(i int, result1 *v1.Resource, result2 error) {
	fake.updateInstanceMutex.Lock()
	defer fake.updateInstanceMutex.Unlock()
	fake.UpdateInstanceStub = nil
	if fake.updateInstanceReturnsOnCall == nil {
		fake.updateInstanceReturnsOnCall = make(map[int]struct {
			result1 *v1.Resource
			result2 error
		})
	}
	fake.updateInstanceReturnsOnCall[i] = struct {
		result1 *v1.Resource
		result2 error
	}{result1, result2}
}

func (fake *FakeResourceServiceInterface) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.addInstanceMutex.RLock()
	defer fake.addInstanceMutex.RUnlock()
	fake.deleteInstanceMutex.RLock()
	defer fake.deleteInstanceMutex.RUnlock()
	fake.updateInstanceMutex.RLock()
	defer fake.updateInstanceMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeResourceServiceInterface) recordInvocation(key string, args []interface{}) {
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
