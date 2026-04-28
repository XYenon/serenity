package subscription

import (
	"net/http"
	"testing"

	"github.com/sagernet/sing/common/json/badjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequestHeaderOptionsAndApply(t *testing.T) {
	rawRequestHeader := &badjson.TypedMap[string, string]{}
	rawRequestHeader.Put("Authorization", "Bearer token")
	rawRequestHeader.Put("X-Subscription", "serenity")
	requestHeader, err := NewRequestHeaderOptions(rawRequestHeader)
	require.NoError(t, err)

	header := make(http.Header)
	applyRequestHeaders(header, requestHeader)

	assert.Equal(t, "Bearer token", header.Get("Authorization"))
	assert.Equal(t, "serenity", header.Get("X-Subscription"))
}

func TestNewRequestHeaderOptionsInvalidName(t *testing.T) {
	requestHeader := &badjson.TypedMap[string, string]{}
	requestHeader.Put("", "value")

	_, err := NewRequestHeaderOptions(requestHeader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing name")
}
