package performance

import (
	"testing"

	"github.com/nginx/agent/v2/src/core"
)

func BenchmarkNewHostInfo(b *testing.B) {
	b.ResetTimer()
	env := &core.EnvironmentType{}
	for i := 0; i < b.N; i++ {
		env.NewHostInfo("", nil, "", false)
	}
}

func BenchmarkProceses(b *testing.B) {
	b.ResetTimer()
	env := &core.EnvironmentType{}
	for i := 0; i < b.N; i++ {
		env.Processes()
	}
}
