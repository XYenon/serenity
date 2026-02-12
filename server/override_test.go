package server

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyOverrides(t *testing.T) {
	rootJSON := `{
		"outbounds": [
			{"tag": "proxy", "type": "shadowsocks"},
			{"tag": "direct", "type": "direct"}
		],
		"route": {
			"rules": [
				{"domain": ["google.com"], "outbound": "proxy", "enabled": false}
			]
		}
	}`
	var root any
	json.Unmarshal([]byte(rootJSON), &root)

	overrides := []string{
		"$.outbounds[0].tag=new-proxy",
		"$.route.rules[0].enabled=true",
		"$.route.rules[0].domain[0]=youtube.com",
	}

	newRoot, err := applyOverrides(root, overrides)
	assert.NoError(t, err)

	m := newRoot.(map[string]any)
	outbounds := m["outbounds"].([]any)
	assert.Equal(t, "new-proxy", outbounds[0].(map[string]any)["tag"])

	route := m["route"].(map[string]any)
	rules := route["rules"].([]any)
	assert.Equal(t, true, rules[0].(map[string]any)["enabled"])
	assert.Equal(t, "youtube.com", rules[0].(map[string]any)["domain"].([]any)[0])
}

func TestApplyOverrides_Create(t *testing.T) {
	var root any // nil root

	overrides := []string{
		"$.log.level=debug",
		"$.outbounds[0].tag=direct",
		"$.outbounds[1].tag=proxy",
	}

	newRoot, err := applyOverrides(root, overrides)
	assert.NoError(t, err)

	m := newRoot.(map[string]any)
	log := m["log"].(map[string]any)
	assert.Equal(t, "debug", log["level"])

	outbounds := m["outbounds"].([]any)
	assert.Len(t, outbounds, 2)
	assert.Equal(t, "direct", outbounds[0].(map[string]any)["tag"])
	assert.Equal(t, "proxy", outbounds[1].(map[string]any)["tag"])
}

func TestApplyOverrides_Types(t *testing.T) {
	var root any = map[string]any{"foo": "bar", "bar": false, "baz": 1, "qux": []any{}}

	overrides := []string{
		"$.foo=123",
		"$.bar=true",
		"$.baz=null",
		"$.qux=[\"a\", \"b\"]",
	}

	newRoot, err := applyOverrides(root, overrides)
	assert.NoError(t, err)

	m := newRoot.(map[string]any)
	assert.Equal(t, float64(123), m["foo"]) // json.Unmarshal unmarshals numbers to float64 by default
	assert.Equal(t, true, m["bar"])
	assert.Nil(t, m["baz"])
	assert.Equal(t, []any{"a", "b"}, m["qux"])
}

func TestApplyOverrides_Errors(t *testing.T) {
	root := map[string]any{"a": 1}

	// Invalid format (no =)
	_, err := applyOverrides(root, []string{"invalid"})
	assert.Error(t, err)

	// Invalid JSONPath
	_, err = applyOverrides(root, []string{"$[=value"})
	assert.Error(t, err)

	// Type mismatch: Index on map
	_, err = applyOverrides(root, []string{"$[0]=val"})
	assert.Error(t, err)

	// Type mismatch: Name on slice
	rootSlice := []any{1}
	_, err = applyOverrides(rootSlice, []string{"$.foo=val"})
	assert.Error(t, err)

	// Complex path not found
	_, err = applyOverrides(root, []string{"$..foo=bar"})
	assert.Error(t, err)
}
