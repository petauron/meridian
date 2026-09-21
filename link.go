package meridian

import (
	"errors"
	"net/url"
	"strings"
)

func parseVLESSLink(raw string) (*url.URL, error) {
	link, err := url.Parse(raw)
	if err != nil || link.Scheme != "vless" || link.User == nil || link.User.Username() == "" || link.Hostname() == "" || link.Port() == "" || link.Query().Get("security") != "reality" || link.Path != "" {
		return nil, errors.New("meridian: a managed VLESS REALITY link is required")
	}
	if _, password := link.User.Password(); password {
		return nil, errors.New("meridian: invalid VLESS identity")
	}
	return link, nil
}

func sameLinkIdentity(left, right string) bool {
	a, err := parseVLESSLink(left)
	if err != nil {
		return false
	}
	b, err := parseVLESSLink(right)
	return err == nil && a.Host == b.Host && a.User.Username() == b.User.Username() && sameRealityTransport(a, b)
}

func sameRealityTransport(left, right *url.URL) bool {
	for _, key := range []string{"security", "type", "sni", "pbk", "sid", "fp", "flow"} {
		if left.Query().Get(key) != right.Query().Get(key) {
			return false
		}
	}
	return true
}

func sameMihomoRealityTransport(proxy map[string]any, link *url.URL) bool {
	query := link.Query()
	reality, _ := proxy["reality-opts"].(map[string]any)
	return proxy["network"] == query.Get("type") && proxy["servername"] == query.Get("sni") &&
		proxy["flow"] == query.Get("flow") && reality["public-key"] == query.Get("pbk") &&
		strings.TrimSpace(query.Get("sid")) == strings.TrimSpace(stringValue(reality["short-id"]))
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
