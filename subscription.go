package meridian

import (
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxSubscriptionBytes = 4 << 20

type PublishedRoute struct {
	Grant            RouteGrant `json:"grant"`
	EntryName        string     `json:"entryName"`
	EgressRegionCode string     `json:"egressRegionCode,omitempty"`
	BaseLink         string     `json:"baseLink"`
	RouteLink        string     `json:"routeLink"`
}

type NativeEntry struct {
	Credential Credential `json:"credential"`
	Link       string     `json:"link"`
}

func RenderLinks(entries []NativeEntry, accountID string, mode PublishingMode, routes []PublishedRoute, encoded bool) ([]byte, error) {
	plain, err := nativeSubscriptionLines(entries, accountID)
	if err != nil {
		return nil, err
	}
	if encoded {
		plain = []byte(base64.StdEncoding.EncodeToString(plain))
	}
	return composeLinks(plain, accountID, mode, routes, encoded)
}

func RenderMihomo(entries []NativeEntry, accountID string, mode PublishingMode, routes []PublishedRoute) ([]byte, error) {
	plain, err := nativeSubscriptionLines(entries, accountID)
	if err != nil {
		return nil, err
	}
	proxies := make([]any, 0, len(entries))
	names := make([]any, 0, len(entries)+1)
	for _, raw := range strings.Split(strings.TrimSpace(string(plain)), "\n") {
		link, err := parseVLESSLink(raw)
		if err != nil {
			return nil, err
		}
		port, err := strconv.Atoi(link.Port())
		if err != nil || port < 1 || port > 65535 {
			return nil, errors.New("meridian: invalid subscription port")
		}
		query := link.Query()
		name := strings.TrimSpace(link.Fragment)
		if name == "" {
			return nil, errors.New("meridian: subscription name is required")
		}
		realityOptions := map[string]any{"public-key": query.Get("pbk"), "short-id": query.Get("sid")}
		if spiderX := query.Get("spx"); spiderX != "" {
			realityOptions["spider-x"] = spiderX
		}
		proxy := map[string]any{
			"name": name, "type": "vless", "server": link.Hostname(), "port": port,
			"uuid": link.User.Username(), "network": query.Get("type"), "tls": true,
			"servername": query.Get("sni"), "flow": query.Get("flow"), "udp": true,
			"reality-opts": realityOptions,
		}
		if fingerprint := query.Get("fp"); fingerprint != "" {
			proxy["client-fingerprint"] = fingerprint
		}
		proxies = append(proxies, proxy)
		names = append(names, name)
	}
	names = append(names, "DIRECT")
	config := map[string]any{
		"proxies":      proxies,
		"proxy-groups": []any{map[string]any{"name": "节点选择", "type": "select", "proxies": names}},
		"rules":        []any{"MATCH,节点选择"},
	}
	native, err := yaml.Marshal(config)
	if err != nil {
		return nil, errors.New("meridian: cannot encode Mihomo subscription")
	}
	return composeMihomo(native, accountID, mode, routes)
}

func nativeSubscriptionLines(entries []NativeEntry, accountID string) ([]byte, error) {
	if !ValidIdentifier(accountID) || len(entries) == 0 || len(entries) > 1024 {
		return nil, errors.New("meridian: invalid native subscription inventory")
	}
	seenNames, seenRoutes := map[string]bool{}, map[string]bool{}
	seenCredentials := map[string]bool{}
	lines := make([]string, 0, len(entries))
	totalBytes := 0
	for _, entry := range entries {
		totalBytes += len(entry.Link)
		if entry.Credential.Validate() != nil || entry.Credential.Kind != NativeCredential || entry.Credential.AccountID != accountID || !entry.Credential.Enabled || seenCredentials[entry.Credential.ID] {
			return nil, errors.New("meridian: invalid native subscription credential")
		}
		link, err := parseVLESSLink(strings.TrimSpace(entry.Link))
		if err != nil {
			return nil, errors.New("meridian: native subscription identity changed")
		}
		port, portErr := strconv.Atoi(link.Port())
		if portErr != nil || port < 1 || port > 65535 || totalBytes > maxSubscriptionBytes || Identity(link.User.Username()) != entry.Credential.Identity || strings.TrimSpace(link.Fragment) == "" || len([]rune(link.Fragment)) > MaxDisplayNameLength || link.Query().Get("sni") == "" || link.Query().Get("pbk") == "" {
			return nil, errors.New("meridian: native subscription identity changed")
		}
		name := strings.TrimSpace(link.Fragment)
		routeKey := link.Host + "\x00" + link.Query().Get("sni") + "\x00" + link.Query().Get("pbk") + "\x00" + link.Query().Get("sid")
		if seenNames[name] || seenRoutes[routeKey] {
			return nil, errors.New("meridian: ambiguous native subscription inventory")
		}
		seenNames[name], seenRoutes[routeKey], seenCredentials[entry.Credential.ID] = true, true, true
		lines = append(lines, link.String())
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

func composeLinks(native []byte, accountID string, mode PublishingMode, routes []PublishedRoute, encoded bool) ([]byte, error) {
	if len(native) > maxSubscriptionBytes || !ValidIdentifier(accountID) || !mode.Valid() {
		return nil, errors.New("meridian: invalid subscription input")
	}
	plain := native
	if encoded {
		var err error
		plain, err = base64.StdEncoding.DecodeString(strings.TrimSpace(string(native)))
		if err != nil {
			return nil, errors.New("meridian: invalid native subscription")
		}
	}
	lines := strings.Split(strings.TrimRight(string(plain), "\r\n"), "\n")
	identities := map[string]bool{}
	for _, line := range lines {
		if link, err := parseVLESSLink(strings.TrimSpace(line)); err == nil {
			identities[link.User.Username()] = true
		}
	}
	for _, item := range routes {
		if err := validatePublishedRouteScope(item, accountID); err != nil {
			return nil, err
		}
		if !item.Grant.Publishable() || !mode.Valid() || !item.Grant.Mode.Valid() {
			continue
		}
		if err := validatePublishedRouteLinks(item); err != nil {
			return nil, err
		}
		base, err := parseVLESSLink(item.BaseLink)
		if err != nil || !identities[base.User.Username()] || !slices.ContainsFunc(lines, func(line string) bool { return sameLinkIdentity(strings.TrimSpace(line), item.BaseLink) }) {
			return nil, errors.New("meridian: route does not belong to this subscription")
		}
		routed, err := parseVLESSLink(item.RouteLink)
		if err != nil || routed.Host != base.Host || !sameRealityTransport(routed, base) || identities[routed.User.Username()] {
			return nil, errors.New("meridian: invalid routed credential")
		}
		routed.Fragment = routeName(item)
		lines = append(lines, routed.String())
		identities[routed.User.Username()] = true
	}
	lines = slices.DeleteFunc(lines, func(line string) bool {
		return slices.ContainsFunc(routes, func(item PublishedRoute) bool {
			return item.Grant.Publishable() && item.Grant.HideNative && sameLinkIdentity(strings.TrimSpace(line), item.BaseLink)
		})
	})
	result := []byte(strings.Join(lines, "\n") + "\n")
	if encoded {
		result = []byte(base64.StdEncoding.EncodeToString(result))
	}
	return result, nil
}

func composeMihomo(native []byte, accountID string, mode PublishingMode, routes []PublishedRoute) ([]byte, error) {
	if len(native) > maxSubscriptionBytes || !ValidIdentifier(accountID) || !mode.Valid() {
		return nil, errors.New("meridian: invalid subscription input")
	}
	var config map[string]any
	if yaml.Unmarshal(native, &config) != nil || config == nil {
		return nil, errors.New("meridian: invalid native Mihomo configuration")
	}
	proxies, ok := config["proxies"].([]any)
	if !ok {
		return nil, errors.New("meridian: native subscription has no proxy inventory")
	}
	groups, ok := config["proxy-groups"].([]any)
	if !ok && config["proxy-groups"] != nil {
		return nil, errors.New("meridian: invalid native proxy groups")
	}
	names := map[string]bool{"DIRECT": true, "REJECT": true}
	for _, values := range [][]any{proxies, groups} {
		for _, value := range values {
			entry, ok := value.(map[string]any)
			if !ok {
				return nil, errors.New("meridian: invalid native subscription entry")
			}
			name, _ := entry["name"].(string)
			if name == "" || names[name] {
				return nil, errors.New("meridian: ambiguous native subscription names")
			}
			names[name] = true
		}
	}
	reserve := func(name string) error {
		if names[name] {
			return errors.New("meridian: managed subscription name conflicts with native configuration")
		}
		names[name] = true
		return nil
	}
	baseProxies := slices.Clone(proxies)
	routeNames := []any{}
	replacements := map[string][]any{}
	for _, item := range routes {
		if err := validatePublishedRouteScope(item, accountID); err != nil {
			return nil, err
		}
		if !item.Grant.Publishable() {
			continue
		}
		if err := validatePublishedRouteLinks(item); err != nil {
			return nil, err
		}
		base, err := parseVLESSLink(item.BaseLink)
		if err != nil {
			return nil, err
		}
		var proxy map[string]any
		for _, value := range baseProxies {
			candidate := value.(map[string]any)
			if candidate["type"] == "vless" && candidate["uuid"] == base.User.Username() && candidate["server"] == base.Hostname() && fmt.Sprint(candidate["port"]) == base.Port() {
				if proxy != nil {
					return nil, errors.New("meridian: ambiguous native entry identity")
				}
				proxy = candidate
			}
		}
		if proxy == nil || proxy["dialer-proxy"] != nil || !sameMihomoRealityTransport(proxy, base) {
			return nil, errors.New("meridian: native entry does not match this account")
		}
		routed, err := parseVLESSLink(item.RouteLink)
		if err != nil || routed.Host != base.Host || !sameRealityTransport(routed, base) || routed.User.Username() == base.User.Username() {
			return nil, errors.New("meridian: invalid routed subscription identity")
		}
		for _, value := range proxies {
			if value.(map[string]any)["uuid"] == routed.User.Username() {
				return nil, errors.New("meridian: duplicated routed subscription identity")
			}
		}
		name := routeName(item)
		if err := reserve(name); err != nil {
			return nil, err
		}
		clone := make(map[string]any, len(proxy))
		for key, value := range proxy {
			clone[key] = value
		}
		clone["name"], clone["uuid"], clone["udp"] = name, routed.User.Username(), false
		proxies = append(proxies, clone)
		routeNames = append(routeNames, name)
		if item.Grant.HideNative {
			original := proxy["name"].(string)
			replacements[original] = append(replacements[original], name)
		}
	}
	for _, value := range groups {
		group := value.(map[string]any)
		if members, ok := group["proxies"].([]any); ok {
			updated := make([]any, 0, len(members))
			for _, member := range members {
				name, ok := member.(string)
				if !ok {
					return nil, errors.New("meridian: invalid native group member")
				}
				if replacement := replacements[name]; len(replacement) > 0 {
					updated = append(updated, replacement...)
				} else {
					updated = append(updated, member)
				}
			}
			group["proxies"] = updated
		}
		if group["type"] == "select" {
			if members, ok := group["proxies"].([]any); ok {
				for _, name := range routeNames {
					if !slices.Contains(members, name) {
						members = append(members, name)
					}
				}
				group["proxies"] = members
			}
		}
	}
	proxies = slices.DeleteFunc(proxies, func(value any) bool {
		name, _ := value.(map[string]any)["name"].(string)
		return len(replacements[name]) > 0
	})
	if rules, ok := config["rules"].([]any); ok {
		for index, value := range rules {
			if rule, ok := value.(string); ok {
				parts := strings.Split(rule, ",")
				target := len(parts) - 1
				if parts[target] == "no-resolve" && target > 0 {
					target--
				}
				if replacement := replacements[parts[target]]; len(replacement) > 0 {
					parts[target] = replacement[0].(string)
					rules[index] = strings.Join(parts, ",")
				}
			}
		}
	}
	config["proxies"], config["proxy-groups"] = proxies, groups
	output, err := yaml.Marshal(config)
	if err != nil {
		return nil, errors.New("meridian: cannot encode Mihomo subscription")
	}
	return output, nil
}

func validatePublishedRouteScope(item PublishedRoute, accountID string) error {
	if item.Grant.Validate() != nil || item.Grant.AccountID != accountID || !validDisplayName(item.EntryName) || item.EgressRegionCode != "" && !validRegionCode(item.EgressRegionCode) {
		return errors.New("meridian: published route scope does not match")
	}
	return nil
}

func validatePublishedRouteLinks(item PublishedRoute) error {
	base, err := parseVLESSLink(item.BaseLink)
	if err != nil || Identity(base.User.Username()) != item.Grant.Base.Identity {
		return errors.New("meridian: native credential changed")
	}
	routed, err := parseVLESSLink(item.RouteLink)
	if err != nil || Identity(routed.User.Username()) != item.Grant.Route.Identity {
		return errors.New("meridian: routed credential changed")
	}
	return nil
}

func routeName(item PublishedRoute) string {
	entry := strings.TrimSpace(item.EntryName)
	entry = strings.TrimLeft(entry, "｜| ")
	prefix := "🔀｜"
	if item.EgressRegionCode != "" {
		prefix = "🔀 " + regionFlag(item.EgressRegionCode) + "｜"
	}
	return prefix + entry
}

func validRegionCode(code string) bool {
	return len(code) == 2 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z'
}

func regionFlag(code string) string {
	return string([]rune{rune(code[0]-'A') + 0x1F1E6, rune(code[1]-'A') + 0x1F1E6})
}
