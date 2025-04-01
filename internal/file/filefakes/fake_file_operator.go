// Code generated by counterfeiter. DO NOT EDIT.
package filefakes

import (
	"context"
	"sync"

	v1 "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

type FakeFileOperator struct {
	ManifestFileStub        func(map[string]*v1.File) (map[string]*v1.File, error)
	manifestFileMutex       sync.RWMutex
	manifestFileArgsForCall []struct {
		arg1 map[string]*v1.File
	}
	manifestFileReturns struct {
		result1 map[string]*v1.File
		result2 error
	}
	manifestFileReturnsOnCall map[int]struct {
		result1 map[string]*v1.File
		result2 error
	}
	UpdateManifestFileStub        func(map[string]*v1.File) error
	updateManifestFileMutex       sync.RWMutex
	updateManifestFileArgsForCall []struct {
		arg1 map[string]*v1.File
	}
	updateManifestFileReturns struct {
		result1 error
	}
	updateManifestFileReturnsOnCall map[int]struct {
		result1 error
	}
	WriteStub        func(context.Context, []byte, *v1.FileMeta) error
	writeMutex       sync.RWMutex
	writeArgsForCall []struct {
		arg1 context.Context
		arg2 []byte
		arg3 *v1.FileMeta
	}
	writeReturns struct {
		result1 error
	}
	writeReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeFileOperator) ManifestFile(arg1 map[string]*v1.File) (map[string]*v1.File, error) {
	fake.manifestFileMutex.Lock()
	ret, specificReturn := fake.manifestFileReturnsOnCall[len(fake.manifestFileArgsForCall)]
	fake.manifestFileArgsForCall = append(fake.manifestFileArgsForCall, struct {
		arg1 map[string]*v1.File
	}{arg1})
	stub := fake.ManifestFileStub
	fakeReturns := fake.manifestFileReturns
	fake.recordInvocation("ManifestFile", []interface{}{arg1})
	fake.manifestFileMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeFileOperator) ManifestFileCallCount() int {
	fake.manifestFileMutex.RLock()
	defer fake.manifestFileMutex.RUnlock()
	return len(fake.manifestFileArgsForCall)
}

func (fake *FakeFileOperator) ManifestFileCalls(stub func(map[string]*v1.File) (map[string]*v1.File, error)) {
	fake.manifestFileMutex.Lock()
	defer fake.manifestFileMutex.Unlock()
	fake.ManifestFileStub = stub
}

func (fake *FakeFileOperator) ManifestFileArgsForCall(i int) map[string]*v1.File {
	fake.manifestFileMutex.RLock()
	defer fake.manifestFileMutex.RUnlock()
	argsForCall := fake.manifestFileArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeFileOperator) ManifestFileReturns(result1 map[string]*v1.File, result2 error) {
	fake.manifestFileMutex.Lock()
	defer fake.manifestFileMutex.Unlock()
	fake.ManifestFileStub = nil
	fake.manifestFileReturns = struct {
		result1 map[string]*v1.File
		result2 error
	}{result1, result2}
}

func (fake *FakeFileOperator) ManifestFileReturnsOnCall(i int, result1 map[string]*v1.File, result2 error) {
	fake.manifestFileMutex.Lock()
	defer fake.manifestFileMutex.Unlock()
	fake.ManifestFileStub = nil
	if fake.manifestFileReturnsOnCall == nil {
		fake.manifestFileReturnsOnCall = make(map[int]struct {
			result1 map[string]*v1.File
			result2 error
		})
	}
	fake.manifestFileReturnsOnCall[i] = struct {
		result1 map[string]*v1.File
		result2 error
	}{result1, result2}
}

func (fake *FakeFileOperator) UpdateManifestFile(arg1 map[string]*v1.File) error {
	fake.updateManifestFileMutex.Lock()
	ret, specificReturn := fake.updateManifestFileReturnsOnCall[len(fake.updateManifestFileArgsForCall)]
	fake.updateManifestFileArgsForCall = append(fake.updateManifestFileArgsForCall, struct {
		arg1 map[string]*v1.File
	}{arg1})
	stub := fake.UpdateManifestFileStub
	fakeReturns := fake.updateManifestFileReturns
	fake.recordInvocation("UpdateManifestFile", []interface{}{arg1})
	fake.updateManifestFileMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeFileOperator) UpdateManifestFileCallCount() int {
	fake.updateManifestFileMutex.RLock()
	defer fake.updateManifestFileMutex.RUnlock()
	return len(fake.updateManifestFileArgsForCall)
}

func (fake *FakeFileOperator) UpdateManifestFileCalls(stub func(map[string]*v1.File) error) {
	fake.updateManifestFileMutex.Lock()
	defer fake.updateManifestFileMutex.Unlock()
	fake.UpdateManifestFileStub = stub
}

func (fake *FakeFileOperator) UpdateManifestFileArgsForCall(i int) map[string]*v1.File {
	fake.updateManifestFileMutex.RLock()
	defer fake.updateManifestFileMutex.RUnlock()
	argsForCall := fake.updateManifestFileArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeFileOperator) UpdateManifestFileReturns(result1 error) {
	fake.updateManifestFileMutex.Lock()
	defer fake.updateManifestFileMutex.Unlock()
	fake.UpdateManifestFileStub = nil
	fake.updateManifestFileReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeFileOperator) UpdateManifestFileReturnsOnCall(i int, result1 error) {
	fake.updateManifestFileMutex.Lock()
	defer fake.updateManifestFileMutex.Unlock()
	fake.UpdateManifestFileStub = nil
	if fake.updateManifestFileReturnsOnCall == nil {
		fake.updateManifestFileReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.updateManifestFileReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeFileOperator) Write(arg1 context.Context, arg2 []byte, arg3 *v1.FileMeta) error {
	var arg2Copy []byte
	if arg2 != nil {
		arg2Copy = make([]byte, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.writeMutex.Lock()
	ret, specificReturn := fake.writeReturnsOnCall[len(fake.writeArgsForCall)]
	fake.writeArgsForCall = append(fake.writeArgsForCall, struct {
		arg1 context.Context
		arg2 []byte
		arg3 *v1.FileMeta
	}{arg1, arg2Copy, arg3})
	stub := fake.WriteStub
	fakeReturns := fake.writeReturns
	fake.recordInvocation("Write", []interface{}{arg1, arg2Copy, arg3})
	fake.writeMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeFileOperator) WriteCallCount() int {
	fake.writeMutex.RLock()
	defer fake.writeMutex.RUnlock()
	return len(fake.writeArgsForCall)
}

func (fake *FakeFileOperator) WriteCalls(stub func(context.Context, []byte, *v1.FileMeta) error) {
	fake.writeMutex.Lock()
	defer fake.writeMutex.Unlock()
	fake.WriteStub = stub
}

func (fake *FakeFileOperator) WriteArgsForCall(i int) (context.Context, []byte, *v1.FileMeta) {
	fake.writeMutex.RLock()
	defer fake.writeMutex.RUnlock()
	argsForCall := fake.writeArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeFileOperator) WriteReturns(result1 error) {
	fake.writeMutex.Lock()
	defer fake.writeMutex.Unlock()
	fake.WriteStub = nil
	fake.writeReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeFileOperator) WriteReturnsOnCall(i int, result1 error) {
	fake.writeMutex.Lock()
	defer fake.writeMutex.Unlock()
	fake.WriteStub = nil
	if fake.writeReturnsOnCall == nil {
		fake.writeReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.writeReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeFileOperator) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.manifestFileMutex.RLock()
	defer fake.manifestFileMutex.RUnlock()
	fake.updateManifestFileMutex.RLock()
	defer fake.updateManifestFileMutex.RUnlock()
	fake.writeMutex.RLock()
	defer fake.writeMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeFileOperator) recordInvocation(key string, args []interface{}) {
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
