package meridian

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderXrayConfigurationProjectsNativeAndFixedCredentials(t *testing.T) {
	nativeID := "00000000-0000-4000-8000-000000000001"
	routeID := "00000000-0000-4000-8000-000000000002"
	native := Credential{ID: "native", AccountID: "account", Kind: NativeCredential, User: "phone", Identity: Identity(nativeID), EntryID: "entry", Enabled: true}
	route := Credential{ID: "route", AccountID: "account", Kind: RouteCredential, User: RouteUser("grant"), Identity: Identity(routeID), EntryID: "entry", EgressID: "egress", Enabled: true}
	grant := RouteGrant{ID: "grant", AccountID: "account", EntryID: "entry", EgressID: "egress", InboundTag: "meridian-entry", Base: native, Route: route, Mode: FixedMode, Enabled: true, DesiredRev: 2, AppliedRev: 1}
	encoded, err := RenderXrayConfiguration(XrayPlan{
		Revision:    2,
		Endpoints:   []RealityEndpoint{{ID: "endpoint", EntryID: "entry", InboundTag: "meridian-entry", ListenPort: 443, AdvertiseHost: "entry.example.com", AdvertisePort: 443, Target: "www.microsoft.com:443", ServerNames: []string{"www.microsoft.com"}, PrivateKey: "private", PublicKey: "public", ShortIDs: []string{"0123456789abcdef"}, Fingerprint: "chrome"}},
		Credentials: []CredentialMaterial{{Credential: native, ProtocolID: nativeID}, {Credential: route, ProtocolID: routeID}},
		Grants:      []RouteGrant{grant},
		Peers:       []RoutePeer{{EgressID: "egress", Address: "100.64.0.10", Port: 1080}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Inbounds  []map[string]any `json:"inbounds"`
		Outbounds []map[string]any `json:"outbounds"`
		Routing   struct {
			Rules []map[string]any `json:"rules"`
		} `json:"routing"`
	}
	if json.Unmarshal(encoded, &config) != nil || len(config.Inbounds) != 2 || len(config.Outbounds) != 3 || len(config.Routing.Rules) != 4 {
		t.Fatalf("unexpected Xray projection: %s", encoded)
	}
	stream := config.Inbounds[1]["streamSettings"].(map[string]any)
	if config.Inbounds[1]["listen"] != "0.0.0.0" || stream["security"] != "reality" || stream["sockopt"].(map[string]any)["acceptProxyProtocol"] != true {
		t.Fatalf("REALITY bridge contract missing: %#v", config.Inbounds[1])
	}
	if config.Routing.Rules[2]["outboundTag"] != routeOutboundTag("grant") || config.Routing.Rules[3]["outboundTag"] != "blocked" {
		t.Fatalf("fixed route is not fail closed: %#v", config.Routing.Rules)
	}
}

func TestRenderXrayConfigurationRejectsOrphanedRouteCredential(t *testing.T) {
	identifier := "00000000-0000-4000-8000-000000000002"
	route := Credential{ID: "route", AccountID: "account", Kind: RouteCredential, User: RouteUser("grant"), Identity: Identity(identifier), EntryID: "entry", EgressID: "egress", Enabled: true}
	_, err := RenderXrayConfiguration(XrayPlan{
		Revision:    1,
		Endpoints:   []RealityEndpoint{{ID: "endpoint", EntryID: "entry", InboundTag: "meridian-entry", ListenPort: 443, AdvertiseHost: "entry.example.com", AdvertisePort: 443, Target: "www.microsoft.com:443", ServerNames: []string{"www.microsoft.com"}, PrivateKey: "private", PublicKey: "public", ShortIDs: []string{"0123456789abcdef"}}},
		Credentials: []CredentialMaterial{{Credential: route, ProtocolID: identifier}},
	})
	if err == nil || !strings.Contains(err.Error(), "orphaned") {
		t.Fatalf("orphaned route credential error = %v", err)
	}
}

func TestLinkForCredentialPreservesSecretIdentity(t *testing.T) {
	identifier := "00000000-0000-4000-8000-000000000001"
	credential := Credential{ID: "native", AccountID: "account", Kind: NativeCredential, User: "phone", Identity: Identity(identifier), EntryID: "entry", Enabled: true}
	link, err := LinkForCredential(
		RealityEndpoint{ID: "endpoint", EntryID: "entry", InboundTag: "meridian-entry", ListenPort: 443, AdvertiseHost: "entry.example.com", AdvertisePort: 443, Target: "www.microsoft.com:443", ServerNames: []string{"www.microsoft.com"}, PrivateKey: "private", PublicKey: "public", ShortIDs: []string{"0123456789abcdef"}},
		CredentialMaterial{Credential: credential, ProtocolID: identifier},
		"🇺🇸｜Entry Alpha",
	)
	if err != nil || !strings.Contains(link, identifier+"@entry.example.com:443") || !strings.HasSuffix(link, "#🇺🇸｜Entry%20Alpha") {
		t.Fatalf("link=%q err=%v", link, err)
	}
}
