package server

import (
	"fmt"
	"strings"

	"github.com/sagernet/sing/common/json"
	"github.com/theory/jsonpath"
	"github.com/theory/jsonpath/spec"
)

func applyOverrides(root any, overrides []string) (any, error) {
	for _, override := range overrides {
		parts := strings.SplitN(override, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid override format: %s", override)
		}
		pathStr := parts[0]
		valueStr := parts[1]

		var value any
		err := json.Unmarshal([]byte(valueStr), &value)
		if err != nil {
			value = valueStr
		}

		path, err := jsonpath.Parse(pathStr)
		if err != nil {
			return nil, err
		}

		locatedNodes := path.SelectLocated(root)
		for _, node := range locatedNodes {
			var err error
			root, err = setNormalizedPath(root, node.Path, value)
			if err != nil {
				return nil, err
			}
		}
	}
	return root, nil
}

func setNormalizedPath(root any, path spec.NormalizedPath, value any) (any, error) {
	if len(path) == 0 {
		return value, nil
	}

	selector := path[0]
	remainingPath := path[1:]

	switch s := selector.(type) {
	case spec.Name:
		m, ok := root.(map[string]any)
		if !ok {
			return root, fmt.Errorf("type mismatch: expected map for path segment %q, got %T", s, root)
		}
		name := string(s)
		if len(remainingPath) == 0 {
			m[name] = value
		} else {
			newVal, err := setNormalizedPath(m[name], remainingPath, value)
			if err != nil {
				return root, err
			}
			m[name] = newVal
		}
	case spec.Index:
		l, ok := root.([]any)
		if !ok {
			return root, fmt.Errorf("type mismatch: expected slice for path segment %d, got %T", int(s), root)
		}
		index := int(s)
		if index < 0 || index >= len(l) {
			return root, fmt.Errorf("index out of bounds: %d", index)
		}
		if len(remainingPath) == 0 {
			l[index] = value
		} else {
			newVal, err := setNormalizedPath(l[index], remainingPath, value)
			if err != nil {
				return root, err
			}
			l[index] = newVal
		}
	}
	return root, nil
}
