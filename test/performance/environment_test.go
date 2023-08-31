package performance

import (
	"context"
	"os"
	"testing"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/stretchr/testify/assert"
)

func BenchmarkNewHostInfo(b *testing.B) {
	b.ResetTimer()
	env := &core.EnvironmentType{}
	for i := 0; i < b.N; i++ {
		env.NewHostInfo("", nil, "", false)
	}
}

func BenchmarkNewHostInfoWithContext(b *testing.B) {
	b.ResetTimer()
	env := &core.EnvironmentType{}
	for i := 0; i < b.N; i++ {
		ctx, cncl := context.WithCancel(context.Background())
		env.NewHostInfoWithContext(ctx, "", nil, "", false)
		cncl()
	}
}

func BenchmarkProceses(b *testing.B) {
	b.ResetTimer()
	env := &core.EnvironmentType{}
	for i := 0; i < b.N; i++ {
		env.Processes()
	}
}

func BenchmarkGetNetworkOverflow(b *testing.B) {
	b.ResetTimer()
	env := &core.EnvironmentType{}
	for i := 0; i < b.N; i++ {
		env.GetNetOverflow()
	}
}

func BenchmarkGetSystemUUID(b *testing.B) {
	b.ResetTimer()
	env := &core.EnvironmentType{}
	for i := 0; i < b.N; i++ {
		env.GetSystemUUID()
	}
}

func BenchmarkGetContainerID(b *testing.B) {
	b.ResetTimer()
	env := &core.EnvironmentType{}
	for i := 0; i < b.N; i++ {
		env.GetContainerID()
	}
}

func BenchmarkIsContainer(b *testing.B) {
	b.ResetTimer()
	env := &core.EnvironmentType{}
	for i := 0; i < b.N; i++ {
		env.IsContainer()
	}
}

func BenchmarkGetHostname(b *testing.B) {
	b.ResetTimer()
	env := &core.EnvironmentType{}
	for i := 0; i < b.N; i++ {
		env.GetHostname()
	}
}

func BenchmarkDiskDevices(b *testing.B) {
	b.ResetTimer()
	env := &core.EnvironmentType{}
	for i := 0; i < b.N; i++ {
		env.DiskDevices()
	}
}

func BenchmarkWriteFiles(b *testing.B) {
	files := []*proto.File{
		{
			Name:        "/tmp/test.conf",
			Contents:    []byte("contents"),
			Permissions: "0644",
		},
	}

	AllowedDirectoriesMap := map[string]struct{}{"/tmp": {}}
	backup, err := sdk.NewConfigApplyWithIgnoreDirectives("", nil, []string{})
	assert.NoError(b, err)
	b.ResetTimer()

	env := &core.EnvironmentType{}
	for i := 0; i < b.N; i++ {
		env.WriteFiles(backup, files, "/tmp", AllowedDirectoriesMap)
		os.Remove("/tmp/test.conf")
	}
}

func BenchmarkFileStat(b *testing.B) {
	tempDir := b.TempDir()
	tempFile := tempDir + "/test.txt"
	err := os.WriteFile(tempFile, []byte("hello"), 0o644)
	assert.NoError(b, err)
	b.ResetTimer()

	env := &core.EnvironmentType{}
	for i := 0; i < b.N; i++ {
		env.FileStat(tempFile)
	}
	defer os.Remove(tempFile)
}
