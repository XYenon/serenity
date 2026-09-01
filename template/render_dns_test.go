package template

import (
	"context"
	"testing"

	M "github.com/sagernet/serenity/common/metadata"
	"github.com/sagernet/serenity/common/semver"
	serenityOption "github.com/sagernet/serenity/option"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/include"
	boxOption "github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"

	"github.com/stretchr/testify/require"
)

func TestRenderDNSUsesCurrentServerFormat(t *testing.T) {
	t.Parallel()
	ctx := include.Context(context.Background())
	version := semver.ParseVersion("1.14.0")
	var options boxOption.Options
	require.NoError(t, Default.renderDNS(ctx, M.Metadata{Version: &version}, &options))

	content, err := json.MarshalContext(ctx, options)
	require.NoError(t, err)
	require.Contains(t, string(content), `"type":"tls"`)
	require.NotContains(t, string(content), `"address":"tls://8.8.8.8"`)

	var parsedOptions boxOption.Options
	require.NoError(t, json.UnmarshalContext(ctx, content, &parsedOptions))
}

func TestRenderVersionSpecificOptionsFromUserAgent(t *testing.T) {
	t.Parallel()
	ctx := include.Context(context.Background())
	renderTemplate := &Template{Template: serenityOption.Template{
		HTTPClients: []boxOption.HTTPClient{{
			Tag: "rule-set",
		}},
		Services: []boxOption.Service{{
			Type:    C.TypeOOMKiller,
			Tag:     "memory",
			Options: &boxOption.OOMKillerServiceOptions{},
		}},
		FindNeighbor:              true,
		DefaultHTTPClient:         "rule-set",
		DisableTrafficBypass:      true,
		DisableTUN:                true,
		DisableSystemProxy:        true,
		DisableCacheFile:          true,
		DisableExternalController: true,
		DisableClashMode:          true,
	}}

	metadata114 := M.Detect("SFA/1.14.0 (sing-box 1.14.0; Clash compatible)")
	require.NotNil(t, metadata114.Version)
	require.Equal(t, "1.14.0", metadata114.Version.String())
	options114, err := renderTemplate.Render(ctx, metadata114, "test", nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, renderTemplate.HTTPClients, options114.HTTPClients)
	require.Equal(t, renderTemplate.Services, options114.Services)
	require.True(t, options114.Route.FindNeighbor)
	require.Equal(t, "rule-set", options114.Route.DefaultHTTPClient)
	content114, err := json.MarshalContext(ctx, options114)
	require.NoError(t, err)
	var parsedOptions114 boxOption.Options
	require.NoError(t, json.UnmarshalContext(ctx, content114, &parsedOptions114))
	require.Len(t, parsedOptions114.HTTPClients, 1)
	require.Equal(t, "rule-set", parsedOptions114.HTTPClients[0].Tag)
	require.Len(t, parsedOptions114.Services, 1)
	require.Equal(t, C.TypeOOMKiller, parsedOptions114.Services[0].Type)
	require.Equal(t, "memory", parsedOptions114.Services[0].Tag)
	require.True(t, parsedOptions114.Route.FindNeighbor)
	require.Equal(t, "rule-set", parsedOptions114.Route.DefaultHTTPClient)

	metadata113 := M.Detect("SFA/1.13.0 (sing-box 1.13.16; Clash compatible)")
	require.NotNil(t, metadata113.Version)
	require.Equal(t, "1.13.16", metadata113.Version.String())
	options113, err := renderTemplate.Render(ctx, metadata113, "test", nil, nil, nil)
	require.NoError(t, err)
	require.Empty(t, options113.HTTPClients)
	require.Empty(t, options113.Services)
	require.False(t, options113.Route.FindNeighbor)
	require.Empty(t, options113.Route.DefaultHTTPClient)
}
