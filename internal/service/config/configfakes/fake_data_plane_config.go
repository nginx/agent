// Code generated by counterfeiter. DO NOT EDIT.
package configfakes

import (
	"context"
	"sync"

	v1 "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	configa "github.com/nginx/agent/v3/internal/datasource/config"
	"github.com/nginx/agent/v3/internal/service/config"
)

type FakeDataPlaneConfig struct {
	ApplyStub        func(context.Context) error
	applyMutex       sync.RWMutex
	applyArgsForCall []struct {
		arg1 context.Context
	}
	applyReturns struct {
		result1 error
	}
	applyReturnsOnCall map[int]struct {
		result1 error
	}
	CompleteStub        func(context.Context) error
	completeMutex       sync.RWMutex
	completeArgsForCall []struct {
		arg1 context.Context
	}
	completeReturns struct {
		result1 error
	}
	completeReturnsOnCall map[int]struct {
		result1 error
	}
	ParseConfigStub        func(context.Context) (any, error)
	parseConfigMutex       sync.RWMutex
	parseConfigArgsForCall []struct {
		arg1 context.Context
	}
	parseConfigReturns struct {
		result1 any
		result2 error
	}
	parseConfigReturnsOnCall map[int]struct {
		result1 any
		result2 error
	}
	RollbackStub        func(context.Context, map[string]*v1.FileMeta, *v1.ManagementPlaneRequest_ConfigApplyRequest, string) error
	rollbackMutex       sync.RWMutex
	rollbackArgsForCall []struct {
		arg1 context.Context
		arg2 map[string]*v1.FileMeta
		arg3 *v1.ManagementPlaneRequest_ConfigApplyRequest
		arg4 string
	}
	rollbackReturns struct {
		result1 error
	}
	rollbackReturnsOnCall map[int]struct {
		result1 error
	}
	SetConfigWriterStub        func(configa.ConfigWriterInterface)
	setConfigWriterMutex       sync.RWMutex
	setConfigWriterArgsForCall []struct {
		arg1 configa.ConfigWriterInterface
	}
	ValidateStub        func(context.Context) error
	validateMutex       sync.RWMutex
	validateArgsForCall []struct {
		arg1 context.Context
	}
	validateReturns struct {
		result1 error
	}
	validateReturnsOnCall map[int]struct {
		result1 error
	}
	WriteStub        func(context.Context, *v1.ManagementPlaneRequest_ConfigApplyRequest) (map[string]*v1.FileMeta, error)
	writeMutex       sync.RWMutex
	writeArgsForCall []struct {
		arg1 context.Context
		arg2 *v1.ManagementPlaneRequest_ConfigApplyRequest
	}
	writeReturns struct {
		result1 map[string]*v1.FileMeta
		result2 error
	}
	writeReturnsOnCall map[int]struct {
		result1 map[string]*v1.FileMeta
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeDataPlaneConfig) Apply(arg1 context.Context) error {
	fake.applyMutex.Lock()
	ret, specificReturn := fake.applyReturnsOnCall[len(fake.applyArgsForCall)]
	fake.applyArgsForCall = append(fake.applyArgsForCall, struct {
		arg1 context.Context
	}{arg1})
	stub := fake.ApplyStub
	fakeReturns := fake.applyReturns
	fake.recordInvocation("Apply", []interface{}{arg1})
	fake.applyMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeDataPlaneConfig) ApplyCallCount() int {
	fake.applyMutex.RLock()
	defer fake.applyMutex.RUnlock()
	return len(fake.applyArgsForCall)
}

func (fake *FakeDataPlaneConfig) ApplyCalls(stub func(context.Context) error) {
	fake.applyMutex.Lock()
	defer fake.applyMutex.Unlock()
	fake.ApplyStub = stub
}

func (fake *FakeDataPlaneConfig) ApplyArgsForCall(i int) context.Context {
	fake.applyMutex.RLock()
	defer fake.applyMutex.RUnlock()
	argsForCall := fake.applyArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeDataPlaneConfig) ApplyReturns(result1 error) {
	fake.applyMutex.Lock()
	defer fake.applyMutex.Unlock()
	fake.ApplyStub = nil
	fake.applyReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeDataPlaneConfig) ApplyReturnsOnCall(i int, result1 error) {
	fake.applyMutex.Lock()
	defer fake.applyMutex.Unlock()
	fake.ApplyStub = nil
	if fake.applyReturnsOnCall == nil {
		fake.applyReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.applyReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeDataPlaneConfig) Complete(arg1 context.Context) error {
	fake.completeMutex.Lock()
	ret, specificReturn := fake.completeReturnsOnCall[len(fake.completeArgsForCall)]
	fake.completeArgsForCall = append(fake.completeArgsForCall, struct {
		arg1 context.Context
	}{arg1})
	stub := fake.CompleteStub
	fakeReturns := fake.completeReturns
	fake.recordInvocation("Complete", []interface{}{arg1})
	fake.completeMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeDataPlaneConfig) CompleteCallCount() int {
	fake.completeMutex.RLock()
	defer fake.completeMutex.RUnlock()
	return len(fake.completeArgsForCall)
}

func (fake *FakeDataPlaneConfig) CompleteCalls(stub func(context.Context) error) {
	fake.completeMutex.Lock()
	defer fake.completeMutex.Unlock()
	fake.CompleteStub = stub
}

func (fake *FakeDataPlaneConfig) CompleteArgsForCall(i int) context.Context {
	fake.completeMutex.RLock()
	defer fake.completeMutex.RUnlock()
	argsForCall := fake.completeArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeDataPlaneConfig) CompleteReturns(result1 error) {
	fake.completeMutex.Lock()
	defer fake.completeMutex.Unlock()
	fake.CompleteStub = nil
	fake.completeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeDataPlaneConfig) CompleteReturnsOnCall(i int, result1 error) {
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

func (fake *FakeDataPlaneConfig) ParseConfig(arg1 context.Context) (any, error) {
	fake.parseConfigMutex.Lock()
	ret, specificReturn := fake.parseConfigReturnsOnCall[len(fake.parseConfigArgsForCall)]
	fake.parseConfigArgsForCall = append(fake.parseConfigArgsForCall, struct {
		arg1 context.Context
	}{arg1})
	stub := fake.ParseConfigStub
	fakeReturns := fake.parseConfigReturns
	fake.recordInvocation("ParseConfig", []interface{}{arg1})
	fake.parseConfigMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeDataPlaneConfig) ParseConfigCallCount() int {
	fake.parseConfigMutex.RLock()
	defer fake.parseConfigMutex.RUnlock()
	return len(fake.parseConfigArgsForCall)
}

func (fake *FakeDataPlaneConfig) ParseConfigCalls(stub func(context.Context) (any, error)) {
	fake.parseConfigMutex.Lock()
	defer fake.parseConfigMutex.Unlock()
	fake.ParseConfigStub = stub
}

func (fake *FakeDataPlaneConfig) ParseConfigArgsForCall(i int) context.Context {
	fake.parseConfigMutex.RLock()
	defer fake.parseConfigMutex.RUnlock()
	argsForCall := fake.parseConfigArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeDataPlaneConfig) ParseConfigReturns(result1 any, result2 error) {
	fake.parseConfigMutex.Lock()
	defer fake.parseConfigMutex.Unlock()
	fake.ParseConfigStub = nil
	fake.parseConfigReturns = struct {
		result1 any
		result2 error
	}{result1, result2}
}

func (fake *FakeDataPlaneConfig) ParseConfigReturnsOnCall(i int, result1 any, result2 error) {
	fake.parseConfigMutex.Lock()
	defer fake.parseConfigMutex.Unlock()
	fake.ParseConfigStub = nil
	if fake.parseConfigReturnsOnCall == nil {
		fake.parseConfigReturnsOnCall = make(map[int]struct {
			result1 any
			result2 error
		})
	}
	fake.parseConfigReturnsOnCall[i] = struct {
		result1 any
		result2 error
	}{result1, result2}
}

func (fake *FakeDataPlaneConfig) Rollback(arg1 context.Context, arg2 map[string]*v1.FileMeta, arg3 *v1.ManagementPlaneRequest_ConfigApplyRequest, arg4 string) error {
	fake.rollbackMutex.Lock()
	ret, specificReturn := fake.rollbackReturnsOnCall[len(fake.rollbackArgsForCall)]
	fake.rollbackArgsForCall = append(fake.rollbackArgsForCall, struct {
		arg1 context.Context
		arg2 map[string]*v1.FileMeta
		arg3 *v1.ManagementPlaneRequest_ConfigApplyRequest
		arg4 string
	}{arg1, arg2, arg3, arg4})
	stub := fake.RollbackStub
	fakeReturns := fake.rollbackReturns
	fake.recordInvocation("Rollback", []interface{}{arg1, arg2, arg3, arg4})
	fake.rollbackMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3, arg4)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeDataPlaneConfig) RollbackCallCount() int {
	fake.rollbackMutex.RLock()
	defer fake.rollbackMutex.RUnlock()
	return len(fake.rollbackArgsForCall)
}

func (fake *FakeDataPlaneConfig) RollbackCalls(stub func(context.Context, map[string]*v1.FileMeta, *v1.ManagementPlaneRequest_ConfigApplyRequest, string) error) {
	fake.rollbackMutex.Lock()
	defer fake.rollbackMutex.Unlock()
	fake.RollbackStub = stub
}

func (fake *FakeDataPlaneConfig) RollbackArgsForCall(i int) (context.Context, map[string]*v1.FileMeta, *v1.ManagementPlaneRequest_ConfigApplyRequest, string) {
	fake.rollbackMutex.RLock()
	defer fake.rollbackMutex.RUnlock()
	argsForCall := fake.rollbackArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4
}

func (fake *FakeDataPlaneConfig) RollbackReturns(result1 error) {
	fake.rollbackMutex.Lock()
	defer fake.rollbackMutex.Unlock()
	fake.RollbackStub = nil
	fake.rollbackReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeDataPlaneConfig) RollbackReturnsOnCall(i int, result1 error) {
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

func (fake *FakeDataPlaneConfig) SetConfigWriter(arg1 configa.ConfigWriterInterface) {
	fake.setConfigWriterMutex.Lock()
	fake.setConfigWriterArgsForCall = append(fake.setConfigWriterArgsForCall, struct {
		arg1 configa.ConfigWriterInterface
	}{arg1})
	stub := fake.SetConfigWriterStub
	fake.recordInvocation("SetConfigWriter", []interface{}{arg1})
	fake.setConfigWriterMutex.Unlock()
	if stub != nil {
		fake.SetConfigWriterStub(arg1)
	}
}

func (fake *FakeDataPlaneConfig) SetConfigWriterCallCount() int {
	fake.setConfigWriterMutex.RLock()
	defer fake.setConfigWriterMutex.RUnlock()
	return len(fake.setConfigWriterArgsForCall)
}

func (fake *FakeDataPlaneConfig) SetConfigWriterCalls(stub func(configa.ConfigWriterInterface)) {
	fake.setConfigWriterMutex.Lock()
	defer fake.setConfigWriterMutex.Unlock()
	fake.SetConfigWriterStub = stub
}

func (fake *FakeDataPlaneConfig) SetConfigWriterArgsForCall(i int) configa.ConfigWriterInterface {
	fake.setConfigWriterMutex.RLock()
	defer fake.setConfigWriterMutex.RUnlock()
	argsForCall := fake.setConfigWriterArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeDataPlaneConfig) Validate(arg1 context.Context) error {
	fake.validateMutex.Lock()
	ret, specificReturn := fake.validateReturnsOnCall[len(fake.validateArgsForCall)]
	fake.validateArgsForCall = append(fake.validateArgsForCall, struct {
		arg1 context.Context
	}{arg1})
	stub := fake.ValidateStub
	fakeReturns := fake.validateReturns
	fake.recordInvocation("Validate", []interface{}{arg1})
	fake.validateMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeDataPlaneConfig) ValidateCallCount() int {
	fake.validateMutex.RLock()
	defer fake.validateMutex.RUnlock()
	return len(fake.validateArgsForCall)
}

func (fake *FakeDataPlaneConfig) ValidateCalls(stub func(context.Context) error) {
	fake.validateMutex.Lock()
	defer fake.validateMutex.Unlock()
	fake.ValidateStub = stub
}

func (fake *FakeDataPlaneConfig) ValidateArgsForCall(i int) context.Context {
	fake.validateMutex.RLock()
	defer fake.validateMutex.RUnlock()
	argsForCall := fake.validateArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeDataPlaneConfig) ValidateReturns(result1 error) {
	fake.validateMutex.Lock()
	defer fake.validateMutex.Unlock()
	fake.ValidateStub = nil
	fake.validateReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeDataPlaneConfig) ValidateReturnsOnCall(i int, result1 error) {
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

func (fake *FakeDataPlaneConfig) Write(arg1 context.Context, arg2 *v1.ManagementPlaneRequest_ConfigApplyRequest) (map[string]*v1.FileMeta, error) {
	fake.writeMutex.Lock()
	ret, specificReturn := fake.writeReturnsOnCall[len(fake.writeArgsForCall)]
	fake.writeArgsForCall = append(fake.writeArgsForCall, struct {
		arg1 context.Context
		arg2 *v1.ManagementPlaneRequest_ConfigApplyRequest
	}{arg1, arg2})
	stub := fake.WriteStub
	fakeReturns := fake.writeReturns
	fake.recordInvocation("Write", []interface{}{arg1, arg2})
	fake.writeMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeDataPlaneConfig) WriteCallCount() int {
	fake.writeMutex.RLock()
	defer fake.writeMutex.RUnlock()
	return len(fake.writeArgsForCall)
}

func (fake *FakeDataPlaneConfig) WriteCalls(stub func(context.Context, *v1.ManagementPlaneRequest_ConfigApplyRequest) (map[string]*v1.FileMeta, error)) {
	fake.writeMutex.Lock()
	defer fake.writeMutex.Unlock()
	fake.WriteStub = stub
}

func (fake *FakeDataPlaneConfig) WriteArgsForCall(i int) (context.Context, *v1.ManagementPlaneRequest_ConfigApplyRequest) {
	fake.writeMutex.RLock()
	defer fake.writeMutex.RUnlock()
	argsForCall := fake.writeArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeDataPlaneConfig) WriteReturns(result1 map[string]*v1.FileMeta, result2 error) {
	fake.writeMutex.Lock()
	defer fake.writeMutex.Unlock()
	fake.WriteStub = nil
	fake.writeReturns = struct {
		result1 map[string]*v1.FileMeta
		result2 error
	}{result1, result2}
}

func (fake *FakeDataPlaneConfig) WriteReturnsOnCall(i int, result1 map[string]*v1.FileMeta, result2 error) {
	fake.writeMutex.Lock()
	defer fake.writeMutex.Unlock()
	fake.WriteStub = nil
	if fake.writeReturnsOnCall == nil {
		fake.writeReturnsOnCall = make(map[int]struct {
			result1 map[string]*v1.FileMeta
			result2 error
		})
	}
	fake.writeReturnsOnCall[i] = struct {
		result1 map[string]*v1.FileMeta
		result2 error
	}{result1, result2}
}

func (fake *FakeDataPlaneConfig) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.applyMutex.RLock()
	defer fake.applyMutex.RUnlock()
	fake.completeMutex.RLock()
	defer fake.completeMutex.RUnlock()
	fake.parseConfigMutex.RLock()
	defer fake.parseConfigMutex.RUnlock()
	fake.rollbackMutex.RLock()
	defer fake.rollbackMutex.RUnlock()
	fake.setConfigWriterMutex.RLock()
	defer fake.setConfigWriterMutex.RUnlock()
	fake.validateMutex.RLock()
	defer fake.validateMutex.RUnlock()
	fake.writeMutex.RLock()
	defer fake.writeMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeDataPlaneConfig) recordInvocation(key string, args []interface{}) {
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

var _ config.DataPlaneConfig = new(FakeDataPlaneConfig)
