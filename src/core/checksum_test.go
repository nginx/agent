package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateNginxID(t *testing.T) {
	result := GenerateNginxID("%s_%s_%s", "/tmp", "/tmp/conf", "nim")
	assert.Equal(t, "5f17da2acca7a4429fd3070b039016360c32b49c0832ed2ced5751d3a1575488", result)
}
