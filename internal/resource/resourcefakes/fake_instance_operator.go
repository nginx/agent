// Code generated by counterfeiter. DO NOT EDIT.
package resourcefakes

import (
	"context"
	"sync"

	v1 "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

type FakeInstanceOperator struct {
	ReloadStub        func(context.Context, *v1.Instance) error
	reloadMutex       sync.RWMutex
	reloadArgsForCall []struct {
		arg1 context.Context
		arg2 *v1.Instance
	}
	reloadReturns struct {
		result1 error
	}
	reloadReturnsOnCall map[int]struct {
		result1 error
	}
	ValidateStub        func(context.Context, *v1.Instance) error
	validateMutex       sync.RWMutex
	validateArgsForCall []struct {
		arg1 context.Context
		arg2 *v1.Instance
	}
	validateReturns struct {
		result1 error
	}
	validateReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeInstanceOperator) Reload(arg1 context.Context, arg2 *v1.Instance) error {
	fake.reloadMutex.Lock()
	ret, specificReturn := fake.reloadReturnsOnCall[len(fake.reloadArgsForCall)]
	fake.reloadArgsForCall = append(fake.reloadArgsForCall, struct {
		arg1 context.Context
		arg2 *v1.Instance
	}{arg1, arg2})
	stub := fake.ReloadStub
	fakeReturns := fake.reloadReturns
	fake.recordInvocation("Reload", []interface{}{arg1, arg2})
	fake.reloadMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeInstanceOperator) ReloadCallCount() int {
	fake.reloadMutex.RLock()
	defer fake.reloadMutex.RUnlock()
	return len(fake.reloadArgsForCall)
}

func (fake *FakeInstanceOperator) ReloadCalls(stub func(context.Context, *v1.Instance) error) {
	fake.reloadMutex.Lock()
	defer fake.reloadMutex.Unlock()
	fake.ReloadStub = stub
}

func (fake *FakeInstanceOperator) ReloadArgsForCall(i int) (context.Context, *v1.Instance) {
	fake.reloadMutex.RLock()
	defer fake.reloadMutex.RUnlock()
	argsForCall := fake.reloadArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeInstanceOperator) ReloadReturns(result1 error) {
	fake.reloadMutex.Lock()
	defer fake.reloadMutex.Unlock()
	fake.ReloadStub = nil
	fake.reloadReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeInstanceOperator) ReloadReturnsOnCall(i int, result1 error) {
	fake.reloadMutex.Lock()
	defer fake.reloadMutex.Unlock()
	fake.ReloadStub = nil
	if fake.reloadReturnsOnCall == nil {
		fake.reloadReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.reloadReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeInstanceOperator) Validate(arg1 context.Context, arg2 *v1.Instance) error {
	fake.validateMutex.Lock()
	ret, specificReturn := fake.validateReturnsOnCall[len(fake.validateArgsForCall)]
	fake.validateArgsForCall = append(fake.validateArgsForCall, struct {
		arg1 context.Context
		arg2 *v1.Instance
	}{arg1, arg2})
	stub := fake.ValidateStub
	fakeReturns := fake.validateReturns
	fake.recordInvocation("Validate", []interface{}{arg1, arg2})
	fake.validateMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeInstanceOperator) ValidateCallCount() int {
	fake.validateMutex.RLock()
	defer fake.validateMutex.RUnlock()
	return len(fake.validateArgsForCall)
}

func (fake *FakeInstanceOperator) ValidateCalls(stub func(context.Context, *v1.Instance) error) {
	fake.validateMutex.Lock()
	defer fake.validateMutex.Unlock()
	fake.ValidateStub = stub
}

func (fake *FakeInstanceOperator) ValidateArgsForCall(i int) (context.Context, *v1.Instance) {
	fake.validateMutex.RLock()
	defer fake.validateMutex.RUnlock()
	argsForCall := fake.validateArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeInstanceOperator) ValidateReturns(result1 error) {
	fake.validateMutex.Lock()
	defer fake.validateMutex.Unlock()
	fake.ValidateStub = nil
	fake.validateReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeInstanceOperator) ValidateReturnsOnCall(i int, result1 error) {
	fake.validateMutex.Lock()
	defer fake.validateMutex.Unlock()
	fake.ValidateStub = nil
	if fake.validateReturnsOnCall == nil {
		fake.validateReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.validateReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeInstanceOperator) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.reloadMutex.RLock()
	defer fake.reloadMutex.RUnlock()
	fake.validateMutex.RLock()
	defer fake.validateMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeInstanceOperator) recordInvocation(key string, args []interface{}) {
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