package template

import (
	"context"
	"net/netip"
	"net/url"

	M "github.com/sagernet/serenity/common/metadata"
	"github.com/sagernet/serenity/common/semver"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json/badoption"
	BM "github.com/sagernet/sing/common/metadata"

	mDNS "github.com/miekg/dns"
)

func (t *Template) renderDNS(_ context.Context, metadata M.Metadata, options *option.Options) error {
	var (
		domainStrategy      option.DomainStrategy
		domainStrategyLocal option.DomainStrategy
	)
	if t.DomainStrategy != option.DomainStrategy(C.DomainStrategyAsIS) {
		domainStrategy = t.DomainStrategy
	} else if t.EnableFakeIP {
		domainStrategy = option.DomainStrategy(C.DomainStrategyPreferIPv4)
	} else {
		domainStrategy = option.DomainStrategy(C.DomainStrategyIPv4Only)
	}
	if t.DomainStrategyLocal != option.DomainStrategy(C.DomainStrategyAsIS) {
		domainStrategyLocal = t.DomainStrategyLocal
	} else {
		domainStrategyLocal = option.DomainStrategy(C.DomainStrategyPreferIPv4)
	}
	if domainStrategyLocal == domainStrategy {
		domainStrategyLocal = 0
	}
	options.DNS = &option.DNSOptions{
		RawDNSOptions: option.RawDNSOptions{
			ReverseMapping: !t.DisableTrafficBypass && metadata.Platform != M.PlatformUnknown && !metadata.Platform.IsApple(),
			DNSClientOptions: option.DNSClientOptions{
				Strategy:         domainStrategy,
				IndependentCache: t.EnableFakeIP,
			},
		},
	}
	dnsDefault := t.DNS
	if dnsDefault == "" {
		dnsDefault = DefaultDNS
	}
	dnsLocal := t.DNSLocal
	if dnsLocal == "" {
		dnsLocal = DefaultDNSLocal
	}
	defaultTag := t.DefaultTag
	if defaultTag == "" {
		defaultTag = DefaultDefaultTag
	}
	defaultDNSResolver := ""
	if dnsDefaultUrl, err := url.Parse(dnsDefault); err == nil && BM.IsDomainName(dnsDefaultUrl.Hostname()) {
		defaultDNSResolver = DNSLocalTag
	}
	defaultDNSOptions, err := createDNSServerOptions(dnsDefault, DNSDefaultTag, defaultTag, defaultDNSResolver)
	if err != nil {
		return E.Cause(err, "create default DNS server")
	}
	options.DNS.Servers = append(options.DNS.Servers, defaultDNSOptions)
	var (
		localDNSIsDomain bool
		localDNSAddress  string
	)
	if t.DisableTrafficBypass {
		localDNSAddress = "local"
	} else {
		localDNSAddress = dnsLocal
		if BM.IsDomainName(dnsLocal) {
			localDNSIsDomain = true
		} else if dnsLocalUrl, err := url.Parse(dnsLocal); err == nil {
			switch dnsLocalUrl.Scheme {
			case "tcp", "udp", "tls", "https", "quic", "h3":
				localDNSIsDomain = true
			}
		}
		if localDNSIsDomain {
			defaultDNSOptions, err = createDNSServerOptions(dnsDefault, DNSDefaultTag, defaultTag, DNSLocalSetupTag)
			if err != nil {
				return E.Cause(err, "create default DNS server")
			}
			options.DNS.Servers[len(options.DNS.Servers)-1] = defaultDNSOptions
		}
	}
	localDNSOptions, err := createDNSServerOptions(localDNSAddress, DNSLocalTag, "", "")
	if err != nil {
		return E.Cause(err, "create local DNS server")
	}
	options.DNS.Servers = append(options.DNS.Servers, localDNSOptions)
	if localDNSIsDomain {
		options.DNS.Servers = append(options.DNS.Servers, option.DNSServerOptions{
			Type:    C.DNSTypeLocal,
			Tag:     DNSLocalSetupTag,
			Options: &option.LocalDNSServerOptions{},
		})
	}
	if t.EnableFakeIP {
		var inet4Range, inet6Range *badoption.Prefix
		if t.CustomFakeIP != nil {
			inet4Range = t.CustomFakeIP.Inet4Range
			inet6Range = t.CustomFakeIP.Inet6Range
		} else {
			inet4Range = (*badoption.Prefix)(common.Ptr(netip.MustParsePrefix("198.18.0.0/15")))
			inet6Range = (*badoption.Prefix)(common.Ptr(netip.MustParsePrefix("fc00::/18")))
		}
		options.DNS.Servers = append(options.DNS.Servers, option.DNSServerOptions{
			Tag:  DNSFakeIPTag,
			Type: C.DNSTypeFakeIP,
			Options: &option.FakeIPDNSServerOptions{
				Inet4Range: inet4Range,
				Inet6Range: inet6Range,
			},
		})
	}
	options.DNS.Servers = append(options.DNS.Servers, t.DNSServers...)
	clashModeRule := t.ClashModeRule
	if clashModeRule == "" {
		clashModeRule = "Rule"
	}
	clashModeGlobal := t.ClashModeGlobal
	if clashModeGlobal == "" {
		clashModeGlobal = "Global"
	}
	clashModeDirect := t.ClashModeDirect
	if clashModeDirect == "" {
		clashModeDirect = "Direct"
	}

	if !t.DisableClashMode {
		options.DNS.Rules = append(options.DNS.Rules, option.DNSRule{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultDNSRule{
				RawDefaultDNSRule: option.RawDefaultDNSRule{
					ClashMode: clashModeGlobal,
				},
				DNSRuleAction: option.DNSRuleAction{
					Action: C.RuleActionTypeRoute,
					RouteOptions: option.DNSRouteActionOptions{
						Server: DNSDefaultTag,
					},
				},
			},
		}, option.DNSRule{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultDNSRule{
				RawDefaultDNSRule: option.RawDefaultDNSRule{
					ClashMode: clashModeDirect,
				},
				DNSRuleAction: option.DNSRuleAction{
					Action: C.RuleActionTypeRoute,
					RouteOptions: option.DNSRouteActionOptions{
						Server: DNSLocalTag,
						AbstractDNSRouteActionOptions: option.AbstractDNSRouteActionOptions{
							Strategy: domainStrategyLocal,
						},
					},
				},
			},
		})
	}
	options.DNS.Rules = append(options.DNS.Rules, t.PreDNSRules...)
	if len(t.CustomDNSRules) == 0 {
		if !t.DisableTrafficBypass {
			options.DNS.Rules = append(options.DNS.Rules, option.DNSRule{
				Type: C.RuleTypeDefault,
				DefaultOptions: option.DefaultDNSRule{
					RawDefaultDNSRule: option.RawDefaultDNSRule{
						RuleSet: []string{"geosite-geolocation-cn"},
					},
					DNSRuleAction: option.DNSRuleAction{
						Action: C.RuleActionTypeRoute,
						RouteOptions: option.DNSRouteActionOptions{
							Server: DNSLocalTag,
							AbstractDNSRouteActionOptions: option.AbstractDNSRouteActionOptions{
								Strategy: domainStrategyLocal,
							},
						},
					},
				},
			})
			if !t.DisableDNSLeak && (metadata.Version == nil || metadata.Version.GreaterThanOrEqual(semver.ParseVersion("1.9.0-alpha.1"))) {
				options.DNS.Rules = append(options.DNS.Rules, option.DNSRule{
					Type: C.RuleTypeDefault,
					DefaultOptions: option.DefaultDNSRule{
						RawDefaultDNSRule: option.RawDefaultDNSRule{
							ClashMode: clashModeRule,
						},
						DNSRuleAction: option.DNSRuleAction{
							Action: C.RuleActionTypeRoute,
							RouteOptions: option.DNSRouteActionOptions{
								Server: DNSDefaultTag,
							},
						},
					},
				}, option.DNSRule{
					Type: C.RuleTypeLogical,
					LogicalOptions: option.LogicalDNSRule{
						RawLogicalDNSRule: option.RawLogicalDNSRule{
							Mode: C.LogicalTypeAnd,
							Rules: []option.DNSRule{
								{
									Type: C.RuleTypeDefault,
									DefaultOptions: option.DefaultDNSRule{
										RawDefaultDNSRule: option.RawDefaultDNSRule{
											RuleSet: []string{"geoip-cn"},
										},
									},
								},
								{
									Type: C.RuleTypeDefault,
									DefaultOptions: option.DefaultDNSRule{
										RawDefaultDNSRule: option.RawDefaultDNSRule{
											RuleSet: []string{"geosite-geolocation-!cn"},
											Invert:  true,
										},
									},
								},
							},
						},
						DNSRuleAction: option.DNSRuleAction{
							Action: C.RuleActionTypeRoute,
							RouteOptions: option.DNSRouteActionOptions{
								Server: DNSLocalTag,
								AbstractDNSRouteActionOptions: option.AbstractDNSRouteActionOptions{
									Strategy: domainStrategyLocal,
								},
							},
						},
					},
				})
			}
		}
	} else {
		options.DNS.Rules = append(options.DNS.Rules, t.CustomDNSRules...)
	}
	if t.EnableFakeIP {
		options.DNS.Rules = append(options.DNS.Rules, option.DNSRule{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultDNSRule{
				RawDefaultDNSRule: option.RawDefaultDNSRule{
					QueryType: []option.DNSQueryType{
						option.DNSQueryType(mDNS.TypeA),
						option.DNSQueryType(mDNS.TypeAAAA),
					},
				},
				DNSRuleAction: option.DNSRuleAction{
					Action: C.RuleActionTypeRoute,
					RouteOptions: option.DNSRouteActionOptions{
						Server: DNSFakeIPTag,
					},
				},
			},
		})
	}
	return nil
}

func createDNSServerOptions(address string, tag string, detour string, resolver string) (option.DNSServerOptions, error) {
	serverURL, err := url.Parse(address)
	if err != nil {
		return option.DNSServerOptions{}, err
	}
	serverType := serverURL.Scheme
	if serverType == "" {
		if address == C.DNSTypeLocal {
			serverType = C.DNSTypeLocal
		} else {
			serverType = C.DNSTypeUDP
		}
	}
	if serverType == C.DNSTypeLocal {
		return option.DNSServerOptions{
			Type:    C.DNSTypeLocal,
			Tag:     tag,
			Options: &option.LocalDNSServerOptions{},
		}, nil
	}
	serverAddress := address
	if serverURL.Scheme != "" {
		serverAddress = serverURL.Host
	}
	serverAddr := BM.ParseSocksaddr(serverAddress)
	if !serverAddr.IsValid() {
		return option.DNSServerOptions{}, E.New("invalid server address: ", address)
	}
	dialerOptions := option.DialerOptions{Detour: detour}
	if resolver != "" {
		dialerOptions.DomainResolver = &option.DomainResolveOptions{Server: resolver}
	}
	remoteOptions := option.RemoteDNSServerOptions{
		RawLocalDNSServerOptions: option.RawLocalDNSServerOptions{DialerOptions: dialerOptions},
		DNSServerAddressOptions: option.DNSServerAddressOptions{
			Server: serverAddr.AddrString(),
		},
	}
	defaultPort := uint16(53)
	switch serverType {
	case C.DNSTypeUDP, C.DNSTypeTCP:
	case C.DNSTypeTLS, C.DNSTypeQUIC:
		defaultPort = 853
	case C.DNSTypeHTTPS, C.DNSTypeHTTP3:
		defaultPort = 443
	default:
		return option.DNSServerOptions{}, E.New("unsupported DNS server scheme: ", serverType)
	}
	if serverAddr.Port != 0 && serverAddr.Port != defaultPort {
		remoteOptions.ServerPort = serverAddr.Port
	}
	var serverOptions any = &remoteOptions
	switch serverType {
	case C.DNSTypeTLS, C.DNSTypeQUIC:
		serverOptions = &option.RemoteTLSDNSServerOptions{RemoteDNSServerOptions: remoteOptions}
	case C.DNSTypeHTTPS, C.DNSTypeHTTP3:
		httpsOptions := &option.RemoteHTTPSDNSServerOptions{
			RemoteTLSDNSServerOptions: option.RemoteTLSDNSServerOptions{RemoteDNSServerOptions: remoteOptions},
		}
		if serverURL.Path != "/dns-query" {
			httpsOptions.Path = serverURL.Path
		}
		serverOptions = httpsOptions
	}
	return option.DNSServerOptions{
		Type:    serverType,
		Tag:     tag,
		Options: serverOptions,
	}, nil
}
