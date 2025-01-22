package credentials

import (
	"github.com/nginx/agent/v3/internal/config"
	"reflect"
	"testing"
)

func TestCredentialWatcherService_NewCredentialWatcherService(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		want   *CredentialWatcherService
	}{
		// TODO: Add test cases.
		{
			name:   "Test 1: ",
			config: &config.Config{},
			want:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCredentialWatcherService(tt.config); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCredentialWatcherService() = %v, want %v", got, tt.want)
			}
		})
	}
}
