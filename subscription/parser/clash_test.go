package parser

import (
	"context"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseClashVLESSReality(t *testing.T) {
	const content = `
proxies:
  - name: vless-reality
    type: vless
    server: example.com
    port: 443
    uuid: 2b24320f-7001-338d-9fb5-399517be9ec7
    udp: true
    network: tcp
    flow: xtls-rprx-vision
    tls: true
    servername: www.apple.com
    client-fingerprint: chrome
    reality-opts:
      public-key: tXGFcWTt1M3qraPQ_XtH7QLlK9rex6UD5ezn9FiQhBE
      short-id: '00000000'
`

	outbounds, err := ParseClashSubscription(context.Background(), content)
	require.NoError(t, err)
	require.Len(t, outbounds, 1)
	assert.Equal(t, C.TypeVLESS, outbounds[0].Type)
	assert.Equal(t, "vless-reality", outbounds[0].Tag)

	vlessOptions, ok := outbounds[0].Options.(*option.VLESSOutboundOptions)
	require.True(t, ok)
	assert.Equal(t, "example.com", vlessOptions.Server)
	assert.Equal(t, uint16(443), vlessOptions.ServerPort)
	assert.Equal(t, "2b24320f-7001-338d-9fb5-399517be9ec7", vlessOptions.UUID)
	assert.Equal(t, "xtls-rprx-vision", vlessOptions.Flow)
	assert.Equal(t, []string{"tcp", "udp"}, vlessOptions.Network.Build())
	require.NotNil(t, vlessOptions.TLS)
	assert.True(t, vlessOptions.TLS.Enabled)
	assert.Equal(t, "www.apple.com", vlessOptions.TLS.ServerName)
	require.NotNil(t, vlessOptions.TLS.UTLS)
	assert.True(t, vlessOptions.TLS.UTLS.Enabled)
	assert.Equal(t, "chrome", vlessOptions.TLS.UTLS.Fingerprint)
	require.NotNil(t, vlessOptions.TLS.Reality)
	assert.True(t, vlessOptions.TLS.Reality.Enabled)
	assert.Equal(t, "tXGFcWTt1M3qraPQ_XtH7QLlK9rex6UD5ezn9FiQhBE", vlessOptions.TLS.Reality.PublicKey)
	assert.Equal(t, "00000000", vlessOptions.TLS.Reality.ShortID)
}
