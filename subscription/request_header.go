package subscription

import (
	"net/http"
	"strings"

	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json/badjson"
)

type RequestHeaderOptions struct {
	header http.Header
}

func NewRequestHeaderOptions(raw *badjson.TypedMap[string, string]) (*RequestHeaderOptions, error) {
	if raw == nil {
		return nil, nil
	}
	header := make(http.Header)
	for headerIndex, entry := range raw.Entries() {
		name := strings.TrimSpace(entry.Key)
		if name == "" {
			return nil, E.New("parse request_header[", headerIndex, "]: missing name")
		}
		header.Set(name, entry.Value)
	}
	return &RequestHeaderOptions{header: header}, nil
}

func applyRequestHeaders(target http.Header, requestHeader *RequestHeaderOptions) {
	if requestHeader == nil {
		return
	}
	for name, values := range requestHeader.header {
		for _, value := range values {
			target.Add(name, value)
		}
	}
}
