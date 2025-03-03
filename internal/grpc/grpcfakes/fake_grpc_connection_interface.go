// Code generated by counterfeiter. DO NOT EDIT.
package grpcfakes

import (
	"context"
	"sync"

	v1 "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/grpc"
)

type FakeGrpcConnectionInterface struct {
	CloseStub        func(context.Context) error
	closeMutex       sync.RWMutex
	closeArgsForCall []struct {
		arg1 context.Context
	}
	closeReturns struct {
		result1 error
	}
	closeReturnsOnCall map[int]struct {
		result1 error
	}
	CommandServiceClientStub        func() v1.CommandServiceClient
	commandServiceClientMutex       sync.RWMutex
	commandServiceClientArgsForCall []struct {
	}
	commandServiceClientReturns struct {
		result1 v1.CommandServiceClient
	}
	commandServiceClientReturnsOnCall map[int]struct {
		result1 v1.CommandServiceClient
	}
	FileServiceClientStub        func() v1.FileServiceClient
	fileServiceClientMutex       sync.RWMutex
	fileServiceClientArgsForCall []struct {
	}
	fileServiceClientReturns struct {
		result1 v1.FileServiceClient
	}
	fileServiceClientReturnsOnCall map[int]struct {
		result1 v1.FileServiceClient
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeGrpcConnectionInterface) Close(arg1 context.Context) error {
	fake.closeMutex.Lock()
	ret, specificReturn := fake.closeReturnsOnCall[len(fake.closeArgsForCall)]
	fake.closeArgsForCall = append(fake.closeArgsForCall, struct {
		arg1 context.Context
	}{arg1})
	stub := fake.CloseStub
	fakeReturns := fake.closeReturns
	fake.recordInvocation("Close", []interface{}{arg1})
	fake.closeMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeGrpcConnectionInterface) CloseCallCount() int {
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	return len(fake.closeArgsForCall)
}

func (fake *FakeGrpcConnectionInterface) CloseCalls(stub func(context.Context) error) {
	fake.closeMutex.Lock()
	defer fake.closeMutex.Unlock()
	fake.CloseStub = stub
}

func (fake *FakeGrpcConnectionInterface) CloseArgsForCall(i int) context.Context {
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	argsForCall := fake.closeArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeGrpcConnectionInterface) CloseReturns(result1 error) {
	fake.closeMutex.Lock()
	defer fake.closeMutex.Unlock()
	fake.CloseStub = nil
	fake.closeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeGrpcConnectionInterface) CloseReturnsOnCall(i int, result1 error) {
	fake.closeMutex.Lock()
	defer fake.closeMutex.Unlock()
	fake.CloseStub = nil
	if fake.closeReturnsOnCall == nil {
		fake.closeReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.closeReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeGrpcConnectionInterface) CommandServiceClient() v1.CommandServiceClient {
	fake.commandServiceClientMutex.Lock()
	ret, specificReturn := fake.commandServiceClientReturnsOnCall[len(fake.commandServiceClientArgsForCall)]
	fake.commandServiceClientArgsForCall = append(fake.commandServiceClientArgsForCall, struct {
	}{})
	stub := fake.CommandServiceClientStub
	fakeReturns := fake.commandServiceClientReturns
	fake.recordInvocation("CommandServiceClient", []interface{}{})
	fake.commandServiceClientMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeGrpcConnectionInterface) CommandServiceClientCallCount() int {
	fake.commandServiceClientMutex.RLock()
	defer fake.commandServiceClientMutex.RUnlock()
	return len(fake.commandServiceClientArgsForCall)
}

func (fake *FakeGrpcConnectionInterface) CommandServiceClientCalls(stub func() v1.CommandServiceClient) {
	fake.commandServiceClientMutex.Lock()
	defer fake.commandServiceClientMutex.Unlock()
	fake.CommandServiceClientStub = stub
}

func (fake *FakeGrpcConnectionInterface) CommandServiceClientReturns(result1 v1.CommandServiceClient) {
	fake.commandServiceClientMutex.Lock()
	defer fake.commandServiceClientMutex.Unlock()
	fake.CommandServiceClientStub = nil
	fake.commandServiceClientReturns = struct {
		result1 v1.CommandServiceClient
	}{result1}
}

func (fake *FakeGrpcConnectionInterface) CommandServiceClientReturnsOnCall(i int, result1 v1.CommandServiceClient) {
	fake.commandServiceClientMutex.Lock()
	defer fake.commandServiceClientMutex.Unlock()
	fake.CommandServiceClientStub = nil
	if fake.commandServiceClientReturnsOnCall == nil {
		fake.commandServiceClientReturnsOnCall = make(map[int]struct {
			result1 v1.CommandServiceClient
		})
	}
	fake.commandServiceClientReturnsOnCall[i] = struct {
		result1 v1.CommandServiceClient
	}{result1}
}

func (fake *FakeGrpcConnectionInterface) FileServiceClient() v1.FileServiceClient {
	fake.fileServiceClientMutex.Lock()
	ret, specificReturn := fake.fileServiceClientReturnsOnCall[len(fake.fileServiceClientArgsForCall)]
	fake.fileServiceClientArgsForCall = append(fake.fileServiceClientArgsForCall, struct {
	}{})
	stub := fake.FileServiceClientStub
	fakeReturns := fake.fileServiceClientReturns
	fake.recordInvocation("FileServiceClient", []interface{}{})
	fake.fileServiceClientMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeGrpcConnectionInterface) FileServiceClientCallCount() int {
	fake.fileServiceClientMutex.RLock()
	defer fake.fileServiceClientMutex.RUnlock()
	return len(fake.fileServiceClientArgsForCall)
}

func (fake *FakeGrpcConnectionInterface) FileServiceClientCalls(stub func() v1.FileServiceClient) {
	fake.fileServiceClientMutex.Lock()
	defer fake.fileServiceClientMutex.Unlock()
	fake.FileServiceClientStub = stub
}

func (fake *FakeGrpcConnectionInterface) FileServiceClientReturns(result1 v1.FileServiceClient) {
	fake.fileServiceClientMutex.Lock()
	defer fake.fileServiceClientMutex.Unlock()
	fake.FileServiceClientStub = nil
	fake.fileServiceClientReturns = struct {
		result1 v1.FileServiceClient
	}{result1}
}

func (fake *FakeGrpcConnectionInterface) FileServiceClientReturnsOnCall(i int, result1 v1.FileServiceClient) {
	fake.fileServiceClientMutex.Lock()
	defer fake.fileServiceClientMutex.Unlock()
	fake.FileServiceClientStub = nil
	if fake.fileServiceClientReturnsOnCall == nil {
		fake.fileServiceClientReturnsOnCall = make(map[int]struct {
			result1 v1.FileServiceClient
		})
	}
	fake.fileServiceClientReturnsOnCall[i] = struct {
		result1 v1.FileServiceClient
	}{result1}
}

func (fake *FakeGrpcConnectionInterface) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	fake.commandServiceClientMutex.RLock()
	defer fake.commandServiceClientMutex.RUnlock()
	fake.fileServiceClientMutex.RLock()
	defer fake.fileServiceClientMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeGrpcConnectionInterface) recordInvocation(key string, args []interface{}) {
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

var _ grpc.GrpcConnectionInterface = new(FakeGrpcConnectionInterface)
