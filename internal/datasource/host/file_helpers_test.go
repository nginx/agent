package host

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestGetPermissions(t *testing.T) {
	file, err := os.CreateTemp(".", "get_permissions_test.txt")
	defer os.Remove(file.Name())
	require.NoError(t, err)

	info, err := os.Stat(file.Name())
	require.NoError(t, err)

	permissions := GetPermissions(info.Mode())

	assert.Equal(t, "0600", permissions)
}
