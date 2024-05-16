// Code generated by counterfeiter. DO NOT EDIT.
package resourcefakes

import (
	"sync"

	v1 "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

type FakeResourceServiceInterface struct {
	AddInstanceStub        func([]*v1.Instance) *v1.Resource
	addInstanceMutex       sync.RWMutex
	addInstanceArgsForCall []struct {
		arg1 []*v1.Instance
	}
	addInstanceReturns struct {
		result1 *v1.Resource
	}
	addInstanceReturnsOnCall map[int]struct {
		result1 *v1.Resource
	}
	DeleteInstanceStub        func([]*v1.Instance) *v1.Resource
	deleteInstanceMutex       sync.RWMutex
	deleteInstanceArgsForCall []struct {
		arg1 []*v1.Instance
	}
	deleteInstanceReturns struct {
		result1 *v1.Resource
	}
	deleteInstanceReturnsOnCall map[int]struct {
		result1 *v1.Resource
	}
	UpdateInstanceStub        func([]*v1.Instance) *v1.Resource
	updateInstanceMutex       sync.RWMutex
	updateInstanceArgsForCall []struct {
		arg1 []*v1.Instance
	}
	updateInstanceReturns struct {
		result1 *v1.Resource
	}
	updateInstanceReturnsOnCall map[int]struct {
		result1 *v1.Resource
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeResourceServiceInterface) AddInstance(arg1 []*v1.Instance) *v1.Resource {
	var arg1Copy []*v1.Instance
	if arg1 != nil {
		arg1Copy = make([]*v1.Instance, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.addInstanceMutex.Lock()
	ret, specificReturn := fake.addInstanceReturnsOnCall[len(fake.addInstanceArgsForCall)]
	fake.addInstanceArgsForCall = append(fake.addInstanceArgsForCall, struct {
		arg1 []*v1.Instance
	}{arg1Copy})
	stub := fake.AddInstanceStub
	fakeReturns := fake.addInstanceReturns
	fake.recordInvocation("AddInstance", []interface{}{arg1Copy})
	fake.addInstanceMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeResourceServiceInterface) AddInstanceCallCount() int {
	fake.addInstanceMutex.RLock()
	defer fake.addInstanceMutex.RUnlock()
	return len(fake.addInstanceArgsForCall)
}

func (fake *FakeResourceServiceInterface) AddInstanceCalls(stub func([]*v1.Instance) *v1.Resource) {
	fake.addInstanceMutex.Lock()
	defer fake.addInstanceMutex.Unlock()
	fake.AddInstanceStub = stub
}

func (fake *FakeResourceServiceInterface) AddInstanceArgsForCall(i int) []*v1.Instance {
	fake.addInstanceMutex.RLock()
	defer fake.addInstanceMutex.RUnlock()
	argsForCall := fake.addInstanceArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeResourceServiceInterface) AddInstanceReturns(result1 *v1.Resource) {
	fake.addInstanceMutex.Lock()
	defer fake.addInstanceMutex.Unlock()
	fake.AddInstanceStub = nil
	fake.addInstanceReturns = struct {
		result1 *v1.Resource
	}{result1}
}

func (fake *FakeResourceServiceInterface) AddInstanceReturnsOnCall(i int, result1 *v1.Resource) {
	fake.addInstanceMutex.Lock()
	defer fake.addInstanceMutex.Unlock()
	fake.AddInstanceStub = nil
	if fake.addInstanceReturnsOnCall == nil {
		fake.addInstanceReturnsOnCall = make(map[int]struct {
			result1 *v1.Resource
		})
	}
	fake.addInstanceReturnsOnCall[i] = struct {
		result1 *v1.Resource
	}{result1}
}

func (fake *FakeResourceServiceInterface) DeleteInstance(arg1 []*v1.Instance) *v1.Resource {
	var arg1Copy []*v1.Instance
	if arg1 != nil {
		arg1Copy = make([]*v1.Instance, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.deleteInstanceMutex.Lock()
	ret, specificReturn := fake.deleteInstanceReturnsOnCall[len(fake.deleteInstanceArgsForCall)]
	fake.deleteInstanceArgsForCall = append(fake.deleteInstanceArgsForCall, struct {
		arg1 []*v1.Instance
	}{arg1Copy})
	stub := fake.DeleteInstanceStub
	fakeReturns := fake.deleteInstanceReturns
	fake.recordInvocation("DeleteInstance", []interface{}{arg1Copy})
	fake.deleteInstanceMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeResourceServiceInterface) DeleteInstanceCallCount() int {
	fake.deleteInstanceMutex.RLock()
	defer fake.deleteInstanceMutex.RUnlock()
	return len(fake.deleteInstanceArgsForCall)
}

func (fake *FakeResourceServiceInterface) DeleteInstanceCalls(stub func([]*v1.Instance) *v1.Resource) {
	fake.deleteInstanceMutex.Lock()
	defer fake.deleteInstanceMutex.Unlock()
	fake.DeleteInstanceStub = stub
}

func (fake *FakeResourceServiceInterface) DeleteInstanceArgsForCall(i int) []*v1.Instance {
	fake.deleteInstanceMutex.RLock()
	defer fake.deleteInstanceMutex.RUnlock()
	argsForCall := fake.deleteInstanceArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeResourceServiceInterface) DeleteInstanceReturns(result1 *v1.Resource) {
	fake.deleteInstanceMutex.Lock()
	defer fake.deleteInstanceMutex.Unlock()
	fake.DeleteInstanceStub = nil
	fake.deleteInstanceReturns = struct {
		result1 *v1.Resource
	}{result1}
}

func (fake *FakeResourceServiceInterface) DeleteInstanceReturnsOnCall(i int, result1 *v1.Resource) {
	fake.deleteInstanceMutex.Lock()
	defer fake.deleteInstanceMutex.Unlock()
	fake.DeleteInstanceStub = nil
	if fake.deleteInstanceReturnsOnCall == nil {
		fake.deleteInstanceReturnsOnCall = make(map[int]struct {
			result1 *v1.Resource
		})
	}
	fake.deleteInstanceReturnsOnCall[i] = struct {
		result1 *v1.Resource
	}{result1}
}

func (fake *FakeResourceServiceInterface) UpdateInstance(arg1 []*v1.Instance) *v1.Resource {
	var arg1Copy []*v1.Instance
	if arg1 != nil {
		arg1Copy = make([]*v1.Instance, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.updateInstanceMutex.Lock()
	ret, specificReturn := fake.updateInstanceReturnsOnCall[len(fake.updateInstanceArgsForCall)]
	fake.updateInstanceArgsForCall = append(fake.updateInstanceArgsForCall, struct {
		arg1 []*v1.Instance
	}{arg1Copy})
	stub := fake.UpdateInstanceStub
	fakeReturns := fake.updateInstanceReturns
	fake.recordInvocation("UpdateInstance", []interface{}{arg1Copy})
	fake.updateInstanceMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeResourceServiceInterface) UpdateInstanceCallCount() int {
	fake.updateInstanceMutex.RLock()
	defer fake.updateInstanceMutex.RUnlock()
	return len(fake.updateInstanceArgsForCall)
}

func (fake *FakeResourceServiceInterface) UpdateInstanceCalls(stub func([]*v1.Instance) *v1.Resource) {
	fake.updateInstanceMutex.Lock()
	defer fake.updateInstanceMutex.Unlock()
	fake.UpdateInstanceStub = stub
}

func (fake *FakeResourceServiceInterface) UpdateInstanceArgsForCall(i int) []*v1.Instance {
	fake.updateInstanceMutex.RLock()
	defer fake.updateInstanceMutex.RUnlock()
	argsForCall := fake.updateInstanceArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeResourceServiceInterface) UpdateInstanceReturns(result1 *v1.Resource) {
	fake.updateInstanceMutex.Lock()
	defer fake.updateInstanceMutex.Unlock()
	fake.UpdateInstanceStub = nil
	fake.updateInstanceReturns = struct {
		result1 *v1.Resource
	}{result1}
}

func (fake *FakeResourceServiceInterface) UpdateInstanceReturnsOnCall(i int, result1 *v1.Resource) {
	fake.updateInstanceMutex.Lock()
	defer fake.updateInstanceMutex.Unlock()
	fake.UpdateInstanceStub = nil
	if fake.updateInstanceReturnsOnCall == nil {
		fake.updateInstanceReturnsOnCall = make(map[int]struct {
			result1 *v1.Resource
		})
	}
	fake.updateInstanceReturnsOnCall[i] = struct {
		result1 *v1.Resource
	}{result1}
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
