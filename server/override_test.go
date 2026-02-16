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

func TestApplyOverrides_FilterSelector(t *testing.T) {
	rootJSON := `{
		"inbounds": [
			{"type": "tun", "tag": "tun-in"},
			{"type": "mixed", "tag": "mixed-in", "listen_port": 8080},
			{"type": "http", "tag": "http-in"}
		]
	}`
	var root any
	json.Unmarshal([]byte(rootJSON), &root)

	// Test filter selector to modify existing property
	// Note: JSONPath uses single quotes for string literals
	overrides := []string{
		`$.inbounds[?(@.type=='mixed')].listen_port=2081`,
	}

	newRoot, err := applyOverrides(root, overrides)
	assert.NoError(t, err)
	assert.NotNil(t, newRoot)

	m := newRoot.(map[string]any)
	inbounds := m["inbounds"].([]any)
	// Only mixed inbound should have listen_port changed
	assert.Equal(t, float64(2081), inbounds[1].(map[string]any)["listen_port"])
	// Other inbounds should remain unchanged
	assert.Nil(t, inbounds[0].(map[string]any)["listen_port"])
	assert.Nil(t, inbounds[2].(map[string]any)["listen_port"])
}

func TestApplyOverrides_FilterSelectorCreateProperty(t *testing.T) {
	rootJSON := `{
		"inbounds": [
			{"type": "tun", "tag": "tun-in"},
			{"type": "mixed", "tag": "mixed-in"},
			{"type": "http", "tag": "http-in"}
		]
	}`
	var root any
	json.Unmarshal([]byte(rootJSON), &root)

	// Test filter selector to create new property
	overrides := []string{
		`$.inbounds[?(@.type=='mixed')].listen_port=2081`,
	}

	newRoot, err := applyOverrides(root, overrides)
	assert.NoError(t, err)
	assert.NotNil(t, newRoot)

	m := newRoot.(map[string]any)
	inbounds := m["inbounds"].([]any)
	// Only mixed inbound should get the new listen_port
	assert.Equal(t, float64(2081), inbounds[1].(map[string]any)["listen_port"])
}

func TestApplyOverrides_WildcardSelector(t *testing.T) {
	rootJSON := `{
		"inbounds": [
			{"type": "mixed", "listen_port": 8080},
			{"type": "mixed", "listen_port": 8081},
			{"type": "http", "listen_port": 8082}
		]
	}`
	var root any
	json.Unmarshal([]byte(rootJSON), &root)

	// Test wildcard selector - modify all listen_port values
	overrides := []string{
		"$.inbounds[*].listen_port=9090",
	}

	newRoot, err := applyOverrides(root, overrides)
	assert.NoError(t, err)

	m := newRoot.(map[string]any)
	inbounds := m["inbounds"].([]any)
	// All inbounds should have listen_port changed
	for i := 0; i < 3; i++ {
		assert.Equal(t, float64(9090), inbounds[i].(map[string]any)["listen_port"])
	}
}

func TestApplyOverrides_SliceSelector(t *testing.T) {
	rootJSON := `{
		"inbounds": [
			{"type": "tun"},
			{"type": "mixed"},
			{"type": "http"},
			{"type": "socks"}
		]
	}`
	var root any
	json.Unmarshal([]byte(rootJSON), &root)

	// Test slice selector - modify first 2 items
	overrides := []string{
		`$.inbounds[0:2].tag=modified`,
	}

	newRoot, err := applyOverrides(root, overrides)
	assert.NoError(t, err)

	m := newRoot.(map[string]any)
	inbounds := m["inbounds"].([]any)
	// First 2 inbounds should have tag
	assert.Equal(t, "modified", inbounds[0].(map[string]any)["tag"])
	assert.Equal(t, "modified", inbounds[1].(map[string]any)["tag"])
	// Last 2 should not have tag
	assert.Nil(t, inbounds[2].(map[string]any)["tag"])
	assert.Nil(t, inbounds[3].(map[string]any)["tag"])
}

func TestApplyOverrides_MultipleSelectors(t *testing.T) {
	rootJSON := `{
		"inbounds": [
			{"type": "mixed", "tag": "in1"},
			{"type": "http", "tag": "in2"},
			{"type": "mixed", "tag": "in3"}
		]
	}`
	var root any
	json.Unmarshal([]byte(rootJSON), &root)

	// Test multiple selectors [0,2]
	overrides := []string{
		`$.inbounds[0,2].listen_port=3000`,
	}

	newRoot, err := applyOverrides(root, overrides)
	assert.NoError(t, err)

	m := newRoot.(map[string]any)
	inbounds := m["inbounds"].([]any)
	assert.Equal(t, float64(3000), inbounds[0].(map[string]any)["listen_port"])
	assert.Nil(t, inbounds[1].(map[string]any)["listen_port"])
	assert.Equal(t, float64(3000), inbounds[2].(map[string]any)["listen_port"])
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
