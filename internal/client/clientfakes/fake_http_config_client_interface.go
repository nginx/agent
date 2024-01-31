// Code generated by counterfeiter. DO NOT EDIT.
package clientfakes

import (
	"sync"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/client"
)

type FakeHttpConfigClientInterface struct {
	GetFileStub        func(*instances.File, string, string) (*instances.FileDownloadResponse, error)
	getFileMutex       sync.RWMutex
	getFileArgsForCall []struct {
		arg1 *instances.File
		arg2 string
		arg3 string
	}
	getFileReturns struct {
		result1 *instances.FileDownloadResponse
		result2 error
	}
	getFileReturnsOnCall map[int]struct {
		result1 *instances.FileDownloadResponse
		result2 error
	}
	GetFilesMetadataStub        func(string, string) (*instances.Files, error)
	getFilesMetadataMutex       sync.RWMutex
	getFilesMetadataArgsForCall []struct {
		arg1 string
		arg2 string
	}
	getFilesMetadataReturns struct {
		result1 *instances.Files
		result2 error
	}
	getFilesMetadataReturnsOnCall map[int]struct {
		result1 *instances.Files
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeHttpConfigClientInterface) GetFile(arg1 *instances.File, arg2 string, arg3 string) (*instances.FileDownloadResponse, error) {
	fake.getFileMutex.Lock()
	ret, specificReturn := fake.getFileReturnsOnCall[len(fake.getFileArgsForCall)]
	fake.getFileArgsForCall = append(fake.getFileArgsForCall, struct {
		arg1 *instances.File
		arg2 string
		arg3 string
	}{arg1, arg2, arg3})
	stub := fake.GetFileStub
	fakeReturns := fake.getFileReturns
	fake.recordInvocation("GetFile", []interface{}{arg1, arg2, arg3})
	fake.getFileMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeHttpConfigClientInterface) GetFileCallCount() int {
	fake.getFileMutex.RLock()
	defer fake.getFileMutex.RUnlock()
	return len(fake.getFileArgsForCall)
}

func (fake *FakeHttpConfigClientInterface) GetFileCalls(stub func(*instances.File, string, string) (*instances.FileDownloadResponse, error)) {
	fake.getFileMutex.Lock()
	defer fake.getFileMutex.Unlock()
	fake.GetFileStub = stub
}

func (fake *FakeHttpConfigClientInterface) GetFileArgsForCall(i int) (*instances.File, string, string) {
	fake.getFileMutex.RLock()
	defer fake.getFileMutex.RUnlock()
	argsForCall := fake.getFileArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeHttpConfigClientInterface) GetFileReturns(result1 *instances.FileDownloadResponse, result2 error) {
	fake.getFileMutex.Lock()
	defer fake.getFileMutex.Unlock()
	fake.GetFileStub = nil
	fake.getFileReturns = struct {
		result1 *instances.FileDownloadResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeHttpConfigClientInterface) GetFileReturnsOnCall(i int, result1 *instances.FileDownloadResponse, result2 error) {
	fake.getFileMutex.Lock()
	defer fake.getFileMutex.Unlock()
	fake.GetFileStub = nil
	if fake.getFileReturnsOnCall == nil {
		fake.getFileReturnsOnCall = make(map[int]struct {
			result1 *instances.FileDownloadResponse
			result2 error
		})
	}
	fake.getFileReturnsOnCall[i] = struct {
		result1 *instances.FileDownloadResponse
		result2 error
	}{result1, result2}
}

func (fake *FakeHttpConfigClientInterface) GetFilesMetadata(arg1 string, arg2 string) (*instances.Files, error) {
	fake.getFilesMetadataMutex.Lock()
	ret, specificReturn := fake.getFilesMetadataReturnsOnCall[len(fake.getFilesMetadataArgsForCall)]
	fake.getFilesMetadataArgsForCall = append(fake.getFilesMetadataArgsForCall, struct {
		arg1 string
		arg2 string
	}{arg1, arg2})
	stub := fake.GetFilesMetadataStub
	fakeReturns := fake.getFilesMetadataReturns
	fake.recordInvocation("GetFilesMetadata", []interface{}{arg1, arg2})
	fake.getFilesMetadataMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeHttpConfigClientInterface) GetFilesMetadataCallCount() int {
	fake.getFilesMetadataMutex.RLock()
	defer fake.getFilesMetadataMutex.RUnlock()
	return len(fake.getFilesMetadataArgsForCall)
}

func (fake *FakeHttpConfigClientInterface) GetFilesMetadataCalls(stub func(string, string) (*instances.Files, error)) {
	fake.getFilesMetadataMutex.Lock()
	defer fake.getFilesMetadataMutex.Unlock()
	fake.GetFilesMetadataStub = stub
}

func (fake *FakeHttpConfigClientInterface) GetFilesMetadataArgsForCall(i int) (string, string) {
	fake.getFilesMetadataMutex.RLock()
	defer fake.getFilesMetadataMutex.RUnlock()
	argsForCall := fake.getFilesMetadataArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeHttpConfigClientInterface) GetFilesMetadataReturns(result1 *instances.Files, result2 error) {
	fake.getFilesMetadataMutex.Lock()
	defer fake.getFilesMetadataMutex.Unlock()
	fake.GetFilesMetadataStub = nil
	fake.getFilesMetadataReturns = struct {
		result1 *instances.Files
		result2 error
	}{result1, result2}
}

func (fake *FakeHttpConfigClientInterface) GetFilesMetadataReturnsOnCall(i int, result1 *instances.Files, result2 error) {
	fake.getFilesMetadataMutex.Lock()
	defer fake.getFilesMetadataMutex.Unlock()
	fake.GetFilesMetadataStub = nil
	if fake.getFilesMetadataReturnsOnCall == nil {
		fake.getFilesMetadataReturnsOnCall = make(map[int]struct {
			result1 *instances.Files
			result2 error
		})
	}
	fake.getFilesMetadataReturnsOnCall[i] = struct {
		result1 *instances.Files
		result2 error
	}{result1, result2}
}

func (fake *FakeHttpConfigClientInterface) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.getFileMutex.RLock()
	defer fake.getFileMutex.RUnlock()
	fake.getFilesMetadataMutex.RLock()
	defer fake.getFilesMetadataMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeHttpConfigClientInterface) recordInvocation(key string, args []interface{}) {
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

var _ client.HttpConfigClientInterface = new(FakeHttpConfigClientInterface)
