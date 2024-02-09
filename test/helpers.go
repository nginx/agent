package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func RemoveFileWithErrorCheck(t *testing.T, fileName string) {
	err := os.Remove(fileName)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("failed on os.Remove of file %s", fileName))
	}
}
