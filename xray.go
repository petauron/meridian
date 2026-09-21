package meridian

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const (
	DefaultRealityPort      = 443
	DefaultLandingSOCKSPort = 1080
	XrayAPIListenPort       = 10085
	XrayMinimumClient       = "0.0.0"
)

type CredentialMaterial struct {
	Credential Credential `json:"credential"`
	ProtocolID string     `json:"protocolId"`
}

func (m CredentialMaterial) Validate() error {
	if m.Credential.Validate() != nil {
		return errors.New("meridian: invalid credential material")
	}
	if _, err := uuid.Parse(m.ProtocolID); err != nil || Identity(m.ProtocolID) != m.Credential.Identity {
		return errors.New("meridian: credential secret does not match its identity")
	}
	return nil
}

type RealityEndpoint struct {
	ID            string   `json:"id"`
	EntryID       string   `json:"entryId"`
	InboundTag    string   `json:"inboundTag"`
	ListenPort    int      `json:"listenPort"`
	AdvertiseHost string   `json:"advertiseHost"`
	AdvertisePort int      `json:"advertisePort"`
	Target        string   `json:"target"`
	ServerNames   []string `json:"serverNames"`
	PrivateKey    string   `json:"privateKey"`
	PublicKey     string   `json:"publicKey"`
	ShortIDs      []string `json:"shortIds"`
	Fingerprint   string   `json:"fingerprint"`
}

func (e RealityEndpoint) Validate() error {
	if !ValidIdentifier(e.ID) || !ValidIdentifier(e.EntryID) || !validInboundTag(e.InboundTag) || !validPort(e.ListenPort) || !validPort(e.AdvertisePort) || !validShareHost(e.AdvertiseHost) || !validRealityTarget(e.Target) || strings.TrimSpace(e.PrivateKey) == "" || strings.TrimSpace(e.PublicKey) == "" || len(e.PrivateKey) > 256 || len(e.PublicKey) > 256 || len(e.ServerNames) == 0 || len(e.ServerNames) > 16 || len(e.ShortIDs) == 0 || len(e.ShortIDs) > 16 {
		return errors.New("meridian: invalid REALITY endpoint")
	}
	seenNames, seenIDs := map[string]bool{}, map[string]bool{}
	for _, name := range e.ServerNames {
		if !validServerName(name) || seenNames[name] {
			return errors.New("meridian: invalid REALITY server name")
		}
		seenNames[name] = true
	}
	for _, shortID := range e.ShortIDs {
		if !validShortID(shortID) || seenIDs[shortID] {
			return errors.New("meridian: invalid REALITY short ID")
		}
		seenIDs[shortID] = true
	}
	if e.Fingerprint != "" && !ValidIdentifier(e.Fingerprint) {
		return errors.New("meridian: invalid REALITY fingerprint")
	}
	return nil
}

type RoutePeer struct {
	EgressID string `json:"egressId"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
}

func (p RoutePeer) Validate() error {
	address, err := netip.ParseAddr(p.Address)
	if !ValidIdentifier(p.EgressID) || err != nil || !privateServiceIPv4(address) || !validPort(p.Port) {
		return errors.New("meridian: invalid route peer")
	}
	return nil
}

type XrayPlan struct {
	Revision    uint64               `json:"revision"`
	Endpoints   []RealityEndpoint    `json:"endpoints"`
	Credentials []CredentialMaterial `json:"credentials"`
	Grants      []RouteGrant         `json:"grants"`
	Peers       []RoutePeer          `json:"peers"`
}

func RenderXrayConfiguration(plan XrayPlan) ([]byte, error) {
	projection, err := prepareXrayProjection(plan)
	if err != nil {
		return nil, err
	}
	inbounds := []any{
		map[string]any{"listen": "127.0.0.1", "port": XrayAPIListenPort, "protocol": "dokodemo-door", "settings": map[string]any{"address": "127.0.0.1"}, "tag": "api"},
	}
	for _, endpoint := range projection.endpoints {
		clients := make([]any, 0, len(projection.credentials[endpoint.EntryID]))
		for _, material := range projection.credentials[endpoint.EntryID] {
			clients = append(clients, map[string]any{
				"email": material.Credential.User,
				"flow":  "xtls-rprx-vision",
				"id":    material.ProtocolID,
				"level": 0,
			})
		}
		inbounds = append(inbounds, map[string]any{
			"listen":   "0.0.0.0",
			"port":     endpoint.ListenPort,
			"protocol": "vless",
			"tag":      endpoint.InboundTag,
			"settings": map[string]any{"clients": clients, "decryption": "none"},
			"streamSettings": map[string]any{
				"network":     "tcp",
				"tcpSettings": map[string]any{"acceptProxyProtocol": true, "header": map[string]any{"type": "none"}},
				"sockopt":     map[string]any{"acceptProxyProtocol": true},
				"security":    "reality",
				"realitySettings": map[string]any{
					"show": false, "xver": 0, "target": endpoint.Target,
					"serverNames": slices.Clone(endpoint.ServerNames), "privateKey": endpoint.PrivateKey,
					"minClientVer": XrayMinimumClient, "maxClientVer": "", "maxTimediff": 0,
					"shortIds": slices.Clone(endpoint.ShortIDs),
				},
			},
		})
	}

	outbounds := []any{map[string]any{"protocol": "freedom", "tag": "direct"}}
	rules := []any{map[string]any{"type": "field", "inboundTag": []string{"api"}, "outboundTag": "api"}}
	for _, grant := range projection.grants {
		peer := projection.peers[grant.EgressID]
		tag := routeOutboundTag(grant.ID)
		outbounds = append(outbounds, map[string]any{
			"protocol": "socks", "tag": tag,
			"settings": map[string]any{"servers": []any{map[string]any{"address": peer.Address, "port": peer.Port}}},
		})
		scope := func(network, outbound string) map[string]any {
			return map[string]any{"type": "field", "inboundTag": []string{grant.InboundTag}, "user": []string{grant.Route.User}, "network": network, "outboundTag": outbound}
		}
		rules = append(rules,
			scope("udp", "blocked"),
			scope("tcp", tag),
			map[string]any{"type": "field", "user": []string{grant.Route.User}, "outboundTag": "blocked"},
		)
	}
	outbounds = append(outbounds, map[string]any{"protocol": "blackhole", "tag": "blocked", "settings": map[string]any{}})
	config := map[string]any{
		"log":       map[string]any{"loglevel": "warning"},
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"api":       map[string]any{"tag": "api", "services": []string{"HandlerService", "LoggerService", "StatsService"}},
		"stats":     map[string]any{},
		"policy": map[string]any{
			"levels": map[string]any{"0": map[string]any{"statsUserUplink": true, "statsUserDownlink": true}},
			"system": map[string]any{"statsInboundUplink": true, "statsInboundDownlink": true},
		},
		"routing": map[string]any{"domainStrategy": "IPIfNonMatch", "rules": rules},
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return nil, errors.New("meridian: cannot encode Xray configuration")
	}
	return encoded, nil
}

func LinkForCredential(endpoint RealityEndpoint, material CredentialMaterial, name string) (string, error) {
	if endpoint.Validate() != nil || material.Validate() != nil || material.Credential.EntryID != endpoint.EntryID || !validDisplayName(name) {
		return "", errors.New("meridian: invalid subscription link input")
	}
	fingerprint := endpoint.Fingerprint
	if fingerprint == "" {
		fingerprint = "chrome"
	}
	query := []string{
		"encryption=none",
		"flow=xtls-rprx-vision",
		"fp=" + fingerprint,
		"pbk=" + endpoint.PublicKey,
		"security=reality",
		"sid=" + endpoint.ShortIDs[0],
		"sni=" + endpoint.ServerNames[0],
		"spx=%2F",
		"type=tcp",
	}
	return "vless://" + material.ProtocolID + "@" + net.JoinHostPort(endpoint.AdvertiseHost, strconv.Itoa(endpoint.AdvertisePort)) + "?" + strings.Join(query, "&") + "#" + escapeLinkFragment(name), nil
}

type xrayProjection struct {
	endpoints   []RealityEndpoint
	credentials map[string][]CredentialMaterial
	grants      []RouteGrant
	peers       map[string]RoutePeer
}

func prepareXrayProjection(plan XrayPlan) (xrayProjection, error) {
	if plan.Revision == 0 || len(plan.Endpoints) == 0 || len(plan.Endpoints) > 128 || len(plan.Credentials) == 0 || len(plan.Credentials) > 65536 || len(plan.Grants) > 65536 || len(plan.Peers) > 1024 {
		return xrayProjection{}, errors.New("meridian: invalid Xray plan")
	}
	projection := xrayProjection{credentials: map[string][]CredentialMaterial{}, peers: map[string]RoutePeer{}}
	entryTags, ports := map[string]string{}, map[int]bool{}
	projection.endpoints = slices.Clone(plan.Endpoints)
	slices.SortFunc(projection.endpoints, func(a, b RealityEndpoint) int { return strings.Compare(a.EntryID, b.EntryID) })
	for _, endpoint := range projection.endpoints {
		if endpoint.Validate() != nil || entryTags[endpoint.EntryID] != "" || ports[endpoint.ListenPort] || endpoint.ListenPort == XrayAPIListenPort {
			return xrayProjection{}, errors.New("meridian: conflicting Xray endpoint")
		}
		entryTags[endpoint.EntryID], ports[endpoint.ListenPort] = endpoint.InboundTag, true
	}
	for _, peer := range plan.Peers {
		if peer.Validate() != nil || projection.peers[peer.EgressID].EgressID != "" {
			return xrayProjection{}, errors.New("meridian: conflicting route peer")
		}
		projection.peers[peer.EgressID] = peer
	}
	materialByID := map[string]CredentialMaterial{}
	users, identities := map[string]bool{}, map[string]bool{}
	for _, material := range plan.Credentials {
		credential := material.Credential
		if material.Validate() != nil || entryTags[credential.EntryID] == "" || materialByID[credential.ID].Credential.ID != "" || users[credential.User] || identities[credential.Identity] {
			return xrayProjection{}, errors.New("meridian: conflicting Xray credential")
		}
		materialByID[credential.ID], users[credential.User], identities[credential.Identity] = material, true, true
		if credential.Enabled {
			projection.credentials[credential.EntryID] = append(projection.credentials[credential.EntryID], material)
		}
	}
	for entryID := range projection.credentials {
		slices.SortFunc(projection.credentials[entryID], func(a, b CredentialMaterial) int { return strings.Compare(a.Credential.ID, b.Credential.ID) })
	}
	projection.grants = slices.Clone(plan.Grants)
	slices.SortFunc(projection.grants, func(a, b RouteGrant) int { return strings.Compare(a.ID, b.ID) })
	seenGrants, usedRoutes := map[string]bool{}, map[string]bool{}
	for _, grant := range projection.grants {
		base, hasBase := materialByID[grant.Base.ID]
		route, hasRoute := materialByID[grant.Route.ID]
		peer, hasPeer := projection.peers[grant.EgressID]
		if !grant.Deployable() || seenGrants[grant.ID] || !hasBase || !hasRoute || base.Credential != grant.Base || route.Credential != grant.Route || usedRoutes[grant.Route.ID] || !hasPeer || peer.EgressID != grant.EgressID || entryTags[grant.EntryID] != grant.InboundTag {
			return xrayProjection{}, errors.New("meridian: conflicting Xray route grant")
		}
		seenGrants[grant.ID], usedRoutes[grant.Route.ID] = true, true
	}
	for _, material := range plan.Credentials {
		if material.Credential.Kind == RouteCredential && material.Credential.Enabled && !usedRoutes[material.Credential.ID] {
			return xrayProjection{}, errors.New("meridian: orphaned routed credential")
		}
	}
	return projection, nil
}

func routeOutboundTag(grantID string) string {
	return "meridian-route-" + Identity(grantID)[:32]
}

func validPort(port int) bool { return port > 0 && port <= 65535 }

func privateServiceIPv4(address netip.Addr) bool {
	cgnat := netip.MustParsePrefix("100.64.0.0/10")
	return address.Is4() && (address.IsPrivate() || cgnat.Contains(address))
}

func validRealityTarget(value string) bool {
	host, port, err := net.SplitHostPort(value)
	if err != nil || !validShareHost(host) {
		return false
	}
	number, err := strconv.Atoi(port)
	return err == nil && validPort(number)
}

func validShareHost(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 253 || strings.ContainsAny(value, "/?#@\x00\r\n") {
		return false
	}
	if net.ParseIP(value) != nil {
		return true
	}
	return validServerName(value)
}

func validServerName(value string) bool {
	if value == "" || len(value) > 253 || strings.Trim(value, ".") != value {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return true
}

func validShortID(value string) bool {
	if len(value) > 16 || len(value)%2 != 0 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}

func escapeLinkFragment(value string) string {
	replacer := strings.NewReplacer("%", "%25", "#", "%23", "?", "%3F", " ", "%20")
	return replacer.Replace(value)
}
