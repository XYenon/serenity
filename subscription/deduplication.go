package subscription

import (
	"context"
	"net/netip"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	boxTLS "github.com/sagernet/sing-box/common/tls"
	C "github.com/sagernet/sing-box/constant"
	boxDNS "github.com/sagernet/sing-box/dns"
	dnsTransport "github.com/sagernet/sing-box/dns/transport"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/common/task"
)

func Deduplication(ctx context.Context, servers []option.Outbound) []option.Outbound {
	logger := log.NewNOPFactory().Logger()
	tlsConfig := common.Must1(boxTLS.NewClient(ctx, logger, "1.1.1.1", option.OutboundTLSOptions{Enabled: true}))
	resolveCtx := &resolveContext{
		ctx: ctx,
		dnsClient: boxDNS.NewClient(boxDNS.ClientOptions{
			Context:       ctx,
			DisableExpire: true,
			ClientSubnet:  netip.MustParsePrefix("114.114.114.114/24"),
			Logger:        logger,
		}),
		dnsTransport: dnsTransport.NewTLSRaw(
			logger,
			boxDNS.NewTransportAdapter(C.DNSTypeTLS, "", nil),
			N.SystemDialer,
			M.ParseSocksaddr("1.1.1.1:853"),
			tlsConfig,
		),
	}

	uniqueServers := make([]netip.AddrPort, len(servers))
	var (
		resolveGroup task.Group
		resultAccess sync.Mutex
	)
	for index, server := range servers {
		currentIndex := index
		currentServer := server
		resolveGroup.Append0(func(ctx context.Context) error {
			destination := resolveDestination(resolveCtx, currentServer)
			if destination.IsValid() {
				resultAccess.Lock()
				uniqueServers[currentIndex] = destination
				resultAccess.Unlock()
			}
			return nil
		})
	}
	resolveGroup.Concurrency(5)
	_ = resolveGroup.Run(ctx)
	uniqueServerMap := make(map[netip.AddrPort]bool)
	var newServers []option.Outbound
	for index, server := range servers {
		destination := uniqueServers[index]
		if destination.IsValid() {
			if uniqueServerMap[destination] {
				continue
			}
			uniqueServerMap[destination] = true
		}
		newServers = append(newServers, server)
	}
	return newServers
}

type resolveContext struct {
	ctx          context.Context
	dnsClient    *boxDNS.Client
	dnsTransport adapter.DNSTransport
}

func resolveDestination(ctx *resolveContext, server option.Outbound) netip.AddrPort {
	serverOptionsWrapper, loaded := server.Options.(option.ServerOptionsWrapper)
	if !loaded {
		return netip.AddrPort{}
	}
	serverOptions := serverOptionsWrapper.TakeServerOptions().Build()
	if serverOptions.IsIP() {
		return serverOptions.AddrPort()
	}
	if serverOptions.IsFqdn() {
		addresses, lookupErr := ctx.dnsClient.Lookup(ctx.ctx, ctx.dnsTransport, serverOptions.Fqdn, adapter.DNSQueryOptions{
			Strategy: C.DomainStrategyPreferIPv4,
		}, nil)
		if lookupErr == nil && len(addresses) > 0 {
			return netip.AddrPortFrom(addresses[0], serverOptions.Port)
		}
	}
	return netip.AddrPort{}
}
