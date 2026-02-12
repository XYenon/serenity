package server

import (
	"errors"
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

		// Try to construct a simple path for creation/modification
		q := path.Query()
		normalizedPath, err := toNormalizedPath(q)
		if err == nil {
			root, err = setNormalizedPath(root, normalizedPath, value)
			if err != nil {
				return nil, err
			}
			continue
		}

		// Fallback to strict existence check for complex paths
		locatedNodes := path.SelectLocated(root)
		if len(locatedNodes) == 0 {
			// For complex paths, if no nodes match, we can't create them ambiguously.
			return nil, fmt.Errorf("path not found: %s", pathStr)
		}

		for _, node := range locatedNodes {
			root, err = setNormalizedPath(root, node.Path, value)
			if err != nil {
				return nil, err
			}
		}
	}
	return root, nil
}

func toNormalizedPath(q *spec.PathQuery) (spec.NormalizedPath, error) {
	segments := q.Segments()
	normalized := make(spec.NormalizedPath, 0, len(segments))
	for _, seg := range segments {
		if seg.IsDescendant() {
			return nil, errors.New("descendant segment not supported")
		}
		selectors := seg.Selectors()
		if len(selectors) != 1 {
			return nil, errors.New("multiple selectors not supported")
		}
		switch s := selectors[0].(type) {
		case spec.Name:
			normalized = append(normalized, s)
		case spec.Index:
			normalized = append(normalized, s)
		default:
			return nil, fmt.Errorf("unsupported selector type: %T", s)
		}
	}
	return normalized, nil
}

func setNormalizedPath(root any, path spec.NormalizedPath, value any) (any, error) {
	if len(path) == 0 {
		return value, nil
	}

	selector := path[0]
	remainingPath := path[1:]

	switch s := selector.(type) {
	case spec.Name:
		name := string(s)
		m, ok := root.(map[string]any)
		if !ok {
			if root == nil {
				m = make(map[string]any)
			} else {
				return nil, fmt.Errorf("expected map for key %s, got %T", name, root)
			}
		}

		var err error
		if len(remainingPath) == 0 {
			m[name] = value
		} else {
			m[name], err = setNormalizedPath(m[name], remainingPath, value)
			if err != nil {
				return nil, err
			}
		}
		return m, nil

	case spec.Index:
		index := int(s)
		l, ok := root.([]any)
		if !ok {
			if root == nil {
				l = make([]any, 0)
			} else {
				return nil, fmt.Errorf("expected slice for index %d, got %T", index, root)
			}
		}

		if index < 0 {
			return nil, fmt.Errorf("index out of bounds: %d", index)
		}
		if index >= len(l) {
			if index > 1024 {
				return nil, fmt.Errorf("index too large: %d", index)
			}
			newSlice := make([]any, index+1)
			copy(newSlice, l)
			l = newSlice
		}

		var err error
		if len(remainingPath) == 0 {
			l[index] = value
		} else {
			l[index], err = setNormalizedPath(l[index], remainingPath, value)
			if err != nil {
				return nil, err
			}
		}
		return l, nil

	default:
		return nil, fmt.Errorf("unknown selector type: %T", selector)
	}
}
