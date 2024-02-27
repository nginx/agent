// Code generated by counterfeiter. DO NOT EDIT.
package configfakes

import (
	"context"
	"sync"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/client"
	"github.com/nginx/agent/v3/internal/datasource/config"
)

type FakeConfigWriterInterface struct {
	CompleteStub        func() error
	completeMutex       sync.RWMutex
	completeArgsForCall []struct {
	}
	completeReturns struct {
		result1 error
	}
	completeReturnsOnCall map[int]struct {
		result1 error
	}
	RollbackStub        func(context.Context, map[string]*instances.File, string, string, string) error
	rollbackMutex       sync.RWMutex
	rollbackArgsForCall []struct {
		arg1 context.Context
		arg2 map[string]*instances.File
		arg3 string
		arg4 string
		arg5 string
	}
	rollbackReturns struct {
		result1 error
	}
	rollbackReturnsOnCall map[int]struct {
		result1 error
	}
	SetConfigClientStub        func(client.ConfigClientInterface)
	setConfigClientMutex       sync.RWMutex
	setConfigClientArgsForCall []struct {
		arg1 client.ConfigClientInterface
	}
	WriteStub        func(context.Context, string, string, string) (map[string]struct{}, error)
	writeMutex       sync.RWMutex
	writeArgsForCall []struct {
		arg1 context.Context
		arg2 string
		arg3 string
		arg4 string
	}
	writeReturns struct {
		result1 map[string]struct{}
		result2 error
	}
	writeReturnsOnCall map[int]struct {
		result1 map[string]struct{}
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeConfigWriterInterface) Complete() error {
	fake.completeMutex.Lock()
	ret, specificReturn := fake.completeReturnsOnCall[len(fake.completeArgsForCall)]
	fake.completeArgsForCall = append(fake.completeArgsForCall, struct {
	}{})
	stub := fake.CompleteStub
	fakeReturns := fake.completeReturns
	fake.recordInvocation("Complete", []interface{}{})
	fake.completeMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeConfigWriterInterface) CompleteCallCount() int {
	fake.completeMutex.RLock()
	defer fake.completeMutex.RUnlock()
	return len(fake.completeArgsForCall)
}

func (fake *FakeConfigWriterInterface) CompleteCalls(stub func() error) {
	fake.completeMutex.Lock()
	defer fake.completeMutex.Unlock()
	fake.CompleteStub = stub
}

func (fake *FakeConfigWriterInterface) CompleteReturns(result1 error) {
	fake.completeMutex.Lock()
	defer fake.completeMutex.Unlock()
	fake.CompleteStub = nil
	fake.completeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeConfigWriterInterface) CompleteReturnsOnCall(i int, result1 error) {
	fake.completeMutex.Lock()
	defer fake.completeMutex.Unlock()
	fake.CompleteStub = nil
	if fake.completeReturnsOnCall == nil {
		fake.completeReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.completeReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeConfigWriterInterface) Rollback(arg1 context.Context, arg2 map[string]*instances.File, arg3 string, arg4 string, arg5 string) error {
	fake.rollbackMutex.Lock()
	ret, specificReturn := fake.rollbackReturnsOnCall[len(fake.rollbackArgsForCall)]
	fake.rollbackArgsForCall = append(fake.rollbackArgsForCall, struct {
		arg1 context.Context
		arg2 map[string]*instances.File
		arg3 string
		arg4 string
		arg5 string
	}{arg1, arg2, arg3, arg4, arg5})
	stub := fake.RollbackStub
	fakeReturns := fake.rollbackReturns
	fake.recordInvocation("Rollback", []interface{}{arg1, arg2, arg3, arg4, arg5})
	fake.rollbackMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3, arg4, arg5)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeConfigWriterInterface) RollbackCallCount() int {
	fake.rollbackMutex.RLock()
	defer fake.rollbackMutex.RUnlock()
	return len(fake.rollbackArgsForCall)
}

func (fake *FakeConfigWriterInterface) RollbackCalls(stub func(context.Context, map[string]*instances.File, string, string, string) error) {
	fake.rollbackMutex.Lock()
	defer fake.rollbackMutex.Unlock()
	fake.RollbackStub = stub
}

func (fake *FakeConfigWriterInterface) RollbackArgsForCall(i int) (context.Context, map[string]*instances.File, string, string, string) {
	fake.rollbackMutex.RLock()
	defer fake.rollbackMutex.RUnlock()
	argsForCall := fake.rollbackArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4, argsForCall.arg5
}

func (fake *FakeConfigWriterInterface) RollbackReturns(result1 error) {
	fake.rollbackMutex.Lock()
	defer fake.rollbackMutex.Unlock()
	fake.RollbackStub = nil
	fake.rollbackReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeConfigWriterInterface) RollbackReturnsOnCall(i int, result1 error) {
	fake.rollbackMutex.Lock()
	defer fake.rollbackMutex.Unlock()
	fake.RollbackStub = nil
	if fake.rollbackReturnsOnCall == nil {
		fake.rollbackReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.rollbackReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeConfigWriterInterface) SetConfigClient(arg1 client.ConfigClientInterface) {
	fake.setConfigClientMutex.Lock()
	fake.setConfigClientArgsForCall = append(fake.setConfigClientArgsForCall, struct {
		arg1 client.ConfigClientInterface
	}{arg1})
	stub := fake.SetConfigClientStub
	fake.recordInvocation("SetConfigClient", []interface{}{arg1})
	fake.setConfigClientMutex.Unlock()
	if stub != nil {
		fake.SetConfigClientStub(arg1)
	}
}

func (fake *FakeConfigWriterInterface) SetConfigClientCallCount() int {
	fake.setConfigClientMutex.RLock()
	defer fake.setConfigClientMutex.RUnlock()
	return len(fake.setConfigClientArgsForCall)
}

func (fake *FakeConfigWriterInterface) SetConfigClientCalls(stub func(client.ConfigClientInterface)) {
	fake.setConfigClientMutex.Lock()
	defer fake.setConfigClientMutex.Unlock()
	fake.SetConfigClientStub = stub
}

func (fake *FakeConfigWriterInterface) SetConfigClientArgsForCall(i int) client.ConfigClientInterface {
	fake.setConfigClientMutex.RLock()
	defer fake.setConfigClientMutex.RUnlock()
	argsForCall := fake.setConfigClientArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeConfigWriterInterface) Write(arg1 context.Context, arg2 string, arg3 string, arg4 string) (map[string]struct{}, error) {
	fake.writeMutex.Lock()
	ret, specificReturn := fake.writeReturnsOnCall[len(fake.writeArgsForCall)]
	fake.writeArgsForCall = append(fake.writeArgsForCall, struct {
		arg1 context.Context
		arg2 string
		arg3 string
		arg4 string
	}{arg1, arg2, arg3, arg4})
	stub := fake.WriteStub
	fakeReturns := fake.writeReturns
	fake.recordInvocation("Write", []interface{}{arg1, arg2, arg3, arg4})
	fake.writeMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3, arg4)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeConfigWriterInterface) WriteCallCount() int {
	fake.writeMutex.RLock()
	defer fake.writeMutex.RUnlock()
	return len(fake.writeArgsForCall)
}

func (fake *FakeConfigWriterInterface) WriteCalls(stub func(context.Context, string, string, string) (map[string]struct{}, error)) {
	fake.writeMutex.Lock()
	defer fake.writeMutex.Unlock()
	fake.WriteStub = stub
}

func (fake *FakeConfigWriterInterface) WriteArgsForCall(i int) (context.Context, string, string, string) {
	fake.writeMutex.RLock()
	defer fake.writeMutex.RUnlock()
	argsForCall := fake.writeArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4
}

func (fake *FakeConfigWriterInterface) WriteReturns(result1 map[string]struct{}, result2 error) {
	fake.writeMutex.Lock()
	defer fake.writeMutex.Unlock()
	fake.WriteStub = nil
	fake.writeReturns = struct {
		result1 map[string]struct{}
		result2 error
	}{result1, result2}
}

func (fake *FakeConfigWriterInterface) WriteReturnsOnCall(i int, result1 map[string]struct{}, result2 error) {
	fake.writeMutex.Lock()
	defer fake.writeMutex.Unlock()
	fake.WriteStub = nil
	if fake.writeReturnsOnCall == nil {
		fake.writeReturnsOnCall = make(map[int]struct {
			result1 map[string]struct{}
			result2 error
		})
	}
	fake.writeReturnsOnCall[i] = struct {
		result1 map[string]struct{}
		result2 error
	}{result1, result2}
}

func (fake *FakeConfigWriterInterface) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.completeMutex.RLock()
	defer fake.completeMutex.RUnlock()
	fake.rollbackMutex.RLock()
	defer fake.rollbackMutex.RUnlock()
	fake.setConfigClientMutex.RLock()
	defer fake.setConfigClientMutex.RUnlock()
	fake.writeMutex.RLock()
	defer fake.writeMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeConfigWriterInterface) recordInvocation(key string, args []interface{}) {
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

var _ config.ConfigWriterInterface = new(FakeConfigWriterInterface)
