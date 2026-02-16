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
		// Use LastIndex to correctly handle filter expressions containing == or =
		eqIdx := strings.LastIndex(override, "=")
		if eqIdx == -1 {
			return nil, fmt.Errorf("invalid override format: %s", override)
		}
		pathStr := override[:eqIdx]
		valueStr := override[eqIdx+1:]

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

		// For complex paths (filter, wildcard, slice, multiple selectors),
		// use the located nodes approach with enhanced path building
		root, err = applyComplexPath(root, q, value)
		if err != nil {
			return nil, err
		}
	}
	return root, nil
}

func applyComplexPath(root any, q *spec.PathQuery, value any) (any, error) {
	segments := q.Segments()
	if len(segments) == 0 {
		return value, nil
	}

	// Build partial path up to the last segment
	parentSegments := segments[:len(segments)-1]
	lastSegment := segments[len(segments)-1]

	// Check if last segment has a single simple selector (and is not descendant)
	// Descendant segments cannot be used to create new properties
	if !lastSegment.IsDescendant() {
		selectors := lastSegment.Selectors()
		if len(selectors) == 1 {
			switch s := selectors[0].(type) {
			case spec.Name:
				root, err := applyToParents(root, parentSegments, spec.NormalSelector(s), value)
				if err != nil {
					return nil, err
				}
				return root, nil
			case spec.Index:
				root, err := applyToParents(root, parentSegments, spec.NormalSelector(s), value)
				if err != nil {
					return nil, err
				}
				return root, nil
			}
		}
	}

	// For complex last segment (filter, wildcard, slice, multiple selectors),
	// we can only modify existing nodes
	locatedNodes := q.SelectLocated(root, root, nil)
	if len(locatedNodes) == 0 {
		return nil, fmt.Errorf("path not found")
	}

	for _, node := range locatedNodes {
		var err error
		root, err = setNormalizedPath(root, node.Path, value)
		if err != nil {
			return nil, err
		}
	}
	return root, nil
}

func applyToParents(root any, parentSegments []*spec.Segment, lastSelector spec.NormalSelector, value any) (any, error) {
	// Build parent path and find parent nodes
	parentQuery := spec.Query(true, parentSegments...)
	parentNodes := parentQuery.SelectLocated(root, root, nil)
	if len(parentNodes) == 0 {
		return nil, fmt.Errorf("path not found")
	}

	// Set value on each parent node
	for _, parentNode := range parentNodes {
		targetPath := make(spec.NormalizedPath, len(parentNode.Path)+1)
		copy(targetPath, parentNode.Path)
		targetPath[len(parentNode.Path)] = lastSelector
		var err error
		root, err = setNormalizedPath(root, targetPath, value)
		if err != nil {
			return nil, err
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
			normalized = append(normalized, spec.NormalSelector(s))
		case spec.Index:
			normalized = append(normalized, spec.NormalSelector(s))
		case *spec.FilterSelector:
			return nil, errors.New("filter selector not supported in normalized path")
		case *spec.WildcardSelector:
			return nil, errors.New("wildcard selector not supported in normalized path")
		case *spec.SliceSelector:
			return nil, errors.New("slice selector not supported in normalized path")
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
