// Code generated by counterfeiter. DO NOT EDIT.
package execfakes

import (
	"bytes"
	"sync"

	"github.com/nginx/agent/v3/internal/datasource/host/exec"
)

type FakeExecInterface struct {
	FindExecutableStub        func(string) (string, error)
	findExecutableMutex       sync.RWMutex
	findExecutableArgsForCall []struct {
		arg1 string
	}
	findExecutableReturns struct {
		result1 string
		result2 error
	}
	findExecutableReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	KillProcessStub        func(int32) error
	killProcessMutex       sync.RWMutex
	killProcessArgsForCall []struct {
		arg1 int32
	}
	killProcessReturns struct {
		result1 error
	}
	killProcessReturnsOnCall map[int]struct {
		result1 error
	}
	RunCmdStub        func(string, ...string) (*bytes.Buffer, error)
	runCmdMutex       sync.RWMutex
	runCmdArgsForCall []struct {
		arg1 string
		arg2 []string
	}
	runCmdReturns struct {
		result1 *bytes.Buffer
		result2 error
	}
	runCmdReturnsOnCall map[int]struct {
		result1 *bytes.Buffer
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeExecInterface) FindExecutable(arg1 string) (string, error) {
	fake.findExecutableMutex.Lock()
	ret, specificReturn := fake.findExecutableReturnsOnCall[len(fake.findExecutableArgsForCall)]
	fake.findExecutableArgsForCall = append(fake.findExecutableArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.FindExecutableStub
	fakeReturns := fake.findExecutableReturns
	fake.recordInvocation("FindExecutable", []interface{}{arg1})
	fake.findExecutableMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeExecInterface) FindExecutableCallCount() int {
	fake.findExecutableMutex.RLock()
	defer fake.findExecutableMutex.RUnlock()
	return len(fake.findExecutableArgsForCall)
}

func (fake *FakeExecInterface) FindExecutableCalls(stub func(string) (string, error)) {
	fake.findExecutableMutex.Lock()
	defer fake.findExecutableMutex.Unlock()
	fake.FindExecutableStub = stub
}

func (fake *FakeExecInterface) FindExecutableArgsForCall(i int) string {
	fake.findExecutableMutex.RLock()
	defer fake.findExecutableMutex.RUnlock()
	argsForCall := fake.findExecutableArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeExecInterface) FindExecutableReturns(result1 string, result2 error) {
	fake.findExecutableMutex.Lock()
	defer fake.findExecutableMutex.Unlock()
	fake.FindExecutableStub = nil
	fake.findExecutableReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeExecInterface) FindExecutableReturnsOnCall(i int, result1 string, result2 error) {
	fake.findExecutableMutex.Lock()
	defer fake.findExecutableMutex.Unlock()
	fake.FindExecutableStub = nil
	if fake.findExecutableReturnsOnCall == nil {
		fake.findExecutableReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.findExecutableReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeExecInterface) KillProcess(arg1 int32) error {
	fake.killProcessMutex.Lock()
	ret, specificReturn := fake.killProcessReturnsOnCall[len(fake.killProcessArgsForCall)]
	fake.killProcessArgsForCall = append(fake.killProcessArgsForCall, struct {
		arg1 int32
	}{arg1})
	stub := fake.KillProcessStub
	fakeReturns := fake.killProcessReturns
	fake.recordInvocation("KillProcess", []interface{}{arg1})
	fake.killProcessMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeExecInterface) KillProcessCallCount() int {
	fake.killProcessMutex.RLock()
	defer fake.killProcessMutex.RUnlock()
	return len(fake.killProcessArgsForCall)
}

func (fake *FakeExecInterface) KillProcessCalls(stub func(int32) error) {
	fake.killProcessMutex.Lock()
	defer fake.killProcessMutex.Unlock()
	fake.KillProcessStub = stub
}

func (fake *FakeExecInterface) KillProcessArgsForCall(i int) int32 {
	fake.killProcessMutex.RLock()
	defer fake.killProcessMutex.RUnlock()
	argsForCall := fake.killProcessArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeExecInterface) KillProcessReturns(result1 error) {
	fake.killProcessMutex.Lock()
	defer fake.killProcessMutex.Unlock()
	fake.KillProcessStub = nil
	fake.killProcessReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeExecInterface) KillProcessReturnsOnCall(i int, result1 error) {
	fake.killProcessMutex.Lock()
	defer fake.killProcessMutex.Unlock()
	fake.KillProcessStub = nil
	if fake.killProcessReturnsOnCall == nil {
		fake.killProcessReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.killProcessReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeExecInterface) RunCmd(arg1 string, arg2 ...string) (*bytes.Buffer, error) {
	fake.runCmdMutex.Lock()
	ret, specificReturn := fake.runCmdReturnsOnCall[len(fake.runCmdArgsForCall)]
	fake.runCmdArgsForCall = append(fake.runCmdArgsForCall, struct {
		arg1 string
		arg2 []string
	}{arg1, arg2})
	stub := fake.RunCmdStub
	fakeReturns := fake.runCmdReturns
	fake.recordInvocation("RunCmd", []interface{}{arg1, arg2})
	fake.runCmdMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeExecInterface) RunCmdCallCount() int {
	fake.runCmdMutex.RLock()
	defer fake.runCmdMutex.RUnlock()
	return len(fake.runCmdArgsForCall)
}

func (fake *FakeExecInterface) RunCmdCalls(stub func(string, ...string) (*bytes.Buffer, error)) {
	fake.runCmdMutex.Lock()
	defer fake.runCmdMutex.Unlock()
	fake.RunCmdStub = stub
}

func (fake *FakeExecInterface) RunCmdArgsForCall(i int) (string, []string) {
	fake.runCmdMutex.RLock()
	defer fake.runCmdMutex.RUnlock()
	argsForCall := fake.runCmdArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeExecInterface) RunCmdReturns(result1 *bytes.Buffer, result2 error) {
	fake.runCmdMutex.Lock()
	defer fake.runCmdMutex.Unlock()
	fake.RunCmdStub = nil
	fake.runCmdReturns = struct {
		result1 *bytes.Buffer
		result2 error
	}{result1, result2}
}

func (fake *FakeExecInterface) RunCmdReturnsOnCall(i int, result1 *bytes.Buffer, result2 error) {
	fake.runCmdMutex.Lock()
	defer fake.runCmdMutex.Unlock()
	fake.RunCmdStub = nil
	if fake.runCmdReturnsOnCall == nil {
		fake.runCmdReturnsOnCall = make(map[int]struct {
			result1 *bytes.Buffer
			result2 error
		})
	}
	fake.runCmdReturnsOnCall[i] = struct {
		result1 *bytes.Buffer
		result2 error
	}{result1, result2}
}

func (fake *FakeExecInterface) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.findExecutableMutex.RLock()
	defer fake.findExecutableMutex.RUnlock()
	fake.killProcessMutex.RLock()
	defer fake.killProcessMutex.RUnlock()
	fake.runCmdMutex.RLock()
	defer fake.runCmdMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeExecInterface) recordInvocation(key string, args []interface{}) {
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

var _ exec.ExecInterface = new(FakeExecInterface)
