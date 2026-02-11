package server

import (
	"github.com/sagernet/sing/common/json"
	"strings"

	"github.com/theory/jsonpath"
	"github.com/theory/jsonpath/spec"
)

func applyOverrides(root any, overrides []string) (any, error) {
	for _, override := range overrides {
		parts := strings.SplitN(override, "=", 2)
		if len(parts) != 2 {
			continue
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
			root = setNormalizedPath(root, node.Path, value)
		}
	}
	return root, nil
}

func setNormalizedPath(root any, path spec.NormalizedPath, value any) any {
	if len(path) == 0 {
		return value
	}

	selector := path[0]
	remainingPath := path[1:]

	switch s := selector.(type) {
	case spec.Name:
		m, ok := root.(map[string]any)
		if !ok {
			return root
		}
		name := string(s)
		if len(remainingPath) == 0 {
			m[name] = value
		} else {
			m[name] = setNormalizedPath(m[name], remainingPath, value)
		}
	case spec.Index:
		l, ok := root.([]any)
		if !ok {
			return root
		}
		index := int(s)
		if index < 0 || index >= len(l) {
			return root
		}
		if len(remainingPath) == 0 {
			l[index] = value
		} else {
			l[index] = setNormalizedPath(l[index], remainingPath, value)
		}
	}
	return root
}
