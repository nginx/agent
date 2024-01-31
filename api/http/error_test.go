package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	err := &RequestError{StatusCode: http.StatusInternalServerError, Message: "something went wrong"}
	assert.Equal(t, "something went wrong", err.Error())
}
