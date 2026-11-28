package noderoute

import (
	"context"
	"encoding/json"
	"net"
	"strings"

	appcommon "github.com/gph-tech/fgmm-strategy-bitfinex/application/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/conf"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/persistent/repository"
)

// RouteMeta parsed from node_route.route_json.
type RouteMeta struct {
	MirrorPlatform       string `json:"mirror_platform"`
	PublishLocalBookTo   string `json:"publish_local_book_to"`
	AcceptMirrorCIDR     string `json:"accept_mirror_cidr"`
	HedgeOrderTo         string `json:"hedge_order_to"`
	TriggersArbOnSrcOnly bool   `json:"triggers_arb_on_src_only"`
}

// Edge one src → dst link (1-of-N).
type Edge struct {
	DstNode        string
	MirrorPlatform string
	PublishTo      string
	HedgeTo        string
}

// Profile loaded for --node != ALL.
type Profile struct {
	NodeCode      string
	LocalPlatform string
	OfficialWS    string
	RelayListen   string
	Edges         []Edge
	byPlatform    map[string]*Edge
}

func (p *Profile) IsLocalPlatform(platform string) bool {
	if p == nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(platform), p.LocalPlatform)
}

// ServesOfficialWS whether this process should dial exchange WS for platform.
func (p *Profile) ServesOfficialWS(platform string) bool {
	if p == nil {
		return true
	}
	return p.IsLocalPlatform(platform)
}

func (p *Profile) EdgeForPlatform(platform string) *Edge {
	if p == nil {
		return nil
	}
	platform = strings.ToUpper(strings.TrimSpace(platform))
	return p.byPlatform[platform]
}

// PlatformForPeerAddr maps TCP remote addr to mirror platform (PublishTo / HedgeTo).
func (p *Profile) PlatformForPeerAddr(remoteAddr string) string {
	if p == nil {
		return ""
	}
	remoteAddr = strings.TrimSpace(remoteAddr)
	remoteHost := peerHost(remoteAddr)
	for _, e := range p.Edges {
		plat := strings.ToUpper(strings.TrimSpace(e.MirrorPlatform))
		if plat == "" {
			continue
		}
		if peerAddrMatch(e.PublishTo, remoteAddr) || peerAddrMatch(e.HedgeTo, remoteAddr) {
			return plat
		}
		if remoteHost != "" && (strings.EqualFold(peerHost(e.PublishTo), remoteHost) ||
			strings.EqualFold(peerHost(e.HedgeTo), remoteHost)) {
			return plat
		}
	}
	return ""
}

func peerHost(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	h, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return h
}

// PeerAddrMatch whether remote TCP addr matches configured host:port (端口须一致)。
func PeerAddrMatch(configured, remote string) bool {
	return peerAddrMatch(configured, remote)
}

func peerAddrMatch(configured, remote string) bool {
	configured = strings.TrimSpace(configured)
	remote = strings.TrimSpace(remote)
	if configured == "" || remote == "" {
		return false
	}
	if strings.EqualFold(configured, remote) {
		return true
	}
	ch, cp, err1 := net.SplitHostPort(configured)
	if err1 != nil {
		ch, cp = configured, "9000"
	}
	rh, rp, err2 := net.SplitHostPort(remote)
	if err2 != nil {
		return false
	}
	return cp == rp && strings.EqualFold(ch, rh)
}

func platformFromDB(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "polymarket", "poly":
		return appcommon.PlatformPolymarket
	case "predict", "pred":
		return appcommon.PlatformPredict
	case "sxbet", "sx":
		return appcommon.PlatformSxbet
	case "kalshi":
		return appcommon.PlatformKalshi
	default:
		return strings.ToUpper(s)
	}
}

// Load builds profile from DB; returns nil, nil when ALL mode.
func Load(ctx context.Context, cfgRepo repository.NodeConfigRepo, routeRepo repository.NodeRouteRepo) (*Profile, error) {
	if conf.IsNodeAllMode() {
		return nil, nil
	}
	code := conf.GetRunNode()
	row, err := cfgRepo.FindByNodeCode(ctx, nil, code)
	if err != nil {
		return nil, err
	}
	routes, err := routeRepo.ListBySrcNode(ctx, nil, code)
	if err != nil {
		return nil, err
	}
	listen := strings.TrimSpace(row.RelayListen)
	if listen == "" {
		listen = ":9000"
	}
	p := &Profile{
		NodeCode:      code,
		LocalPlatform: platformFromDB(row.LocalPlatform),
		OfficialWS:    strings.TrimSpace(row.OfficialWS),
		RelayListen:   listen,
		byPlatform:    map[string]*Edge{},
	}
	for _, r := range routes {
		var meta RouteMeta
		if len(r.RouteJSON) > 0 {
			_ = json.Unmarshal(r.RouteJSON, &meta)
		}
		plat := platformFromDB(meta.MirrorPlatform)
		if plat == "" {
			continue
		}
		pub := strings.TrimSpace(meta.PublishLocalBookTo)
		hedge := strings.TrimSpace(meta.HedgeOrderTo)
		if hedge == "" {
			hedge = pub
		}
		e := Edge{
			DstNode:        strings.TrimSpace(r.DstNode),
			MirrorPlatform: plat,
			PublishTo:      pub,
			HedgeTo:        hedge,
		}
		p.Edges = append(p.Edges, e)
		p.byPlatform[plat] = &p.Edges[len(p.Edges)-1]
	}
	return p, nil
}