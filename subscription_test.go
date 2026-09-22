package meridian

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestRenderLinksPublishesHealthyRouteWithoutLetterSuffix(t *testing.T) {
	baseSecret := "11111111-1111-4111-8111-111111111111"
	routeSecret := "22222222-2222-4222-8222-222222222222"
	baseLink := "vless://" + baseSecret + "@entry.example.com:443?security=reality&type=tcp&sni=www.example.com&pbk=public&sid=abcd&fp=chrome&flow=xtls-rprx-vision#🇺🇸｜Entry-Alpha"
	routeLink := "vless://" + routeSecret + "@entry.example.com:443?security=reality&type=tcp&sni=www.example.com&pbk=public&sid=abcd&fp=chrome&flow=xtls-rprx-vision#ignored"
	grant := RouteGrant{
		ID: "grant-a", AccountID: "account-a", EntryID: "entry-a", EgressID: "egress-a", InboundTag: "vless-a",
		Base:  Credential{ID: "base-a", AccountID: "account-a", Kind: NativeCredential, User: "Phone", Identity: Identity(baseSecret), EntryID: "entry-a", Enabled: true},
		Route: Credential{ID: "route-a", AccountID: "account-a", Kind: RouteCredential, User: RouteUser("grant-a"), Identity: Identity(routeSecret), EntryID: "entry-a", EgressID: "egress-a", Enabled: true},
		Mode:  FixedMode, Enabled: true, DesiredRev: 7, AppliedRev: 7, RuntimeGood: true,
	}
	baseMaterial := CredentialMaterial{Credential: grant.Base, ProtocolID: baseSecret}
	entries := []NativeEntry{{Material: baseMaterial, Protocol: VLESSReality, Link: baseLink}}
	result, err := RenderLinks(entries, "account-a", FixedMode, []PublishedRoute{{Grant: grant, Protocol: VLESSReality, EntryName: "｜Entry-Alpha", EgressRegionCode: "TW", BaseLink: baseLink, RouteLink: routeLink, BaseProtocolIdentity: grant.Base.Identity, RouteProtocolIdentity: grant.Route.Identity}}, true)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(string(result))
	if err != nil {
		t.Fatal(err)
	}
	text := string(decoded)
	if !strings.Contains(text, "🔀 🇹🇼｜Entry-Alpha") || strings.Contains(text, "Entry-Alpha A") || strings.Contains(text, "｜｜") {
		t.Fatalf("rendered subscription = %q", text)
	}
}

func TestRenderLinksSupportsNativeOnlyAccount(t *testing.T) {
	secret := "11111111-1111-4111-8111-111111111111"
	link := "vless://" + secret + "@entry.example.com:443?security=reality&type=tcp&sni=www.example.com&pbk=public&sid=abcd&fp=chrome&flow=xtls-rprx-vision#🇺🇸｜Entry-Alpha"
	credential := Credential{ID: "native-a", AccountID: "account-a", Kind: NativeCredential, User: "Phone", Identity: Identity(secret), EntryID: "entry-a", Enabled: true}
	result, err := RenderLinks([]NativeEntry{{Material: CredentialMaterial{Credential: credential, ProtocolID: secret}, Protocol: VLESSReality, Link: link}}, "account-a", FixedMode, nil, false)
	if err != nil || string(result) != link+"\n" {
		t.Fatalf("native subscription = %q, err=%v", string(result), err)
	}
}

func TestRenderLinksSkipsUnappliedRouteWithoutBreakingNativeSubscription(t *testing.T) {
	secret := "11111111-1111-4111-8111-111111111111"
	routeSecret := "22222222-2222-4222-8222-222222222222"
	link := "vless://" + secret + "@entry.example.com:443?security=reality&type=tcp&sni=www.example.com&pbk=public&sid=abcd&fp=chrome&flow=xtls-rprx-vision#🇺🇸｜Entry-Alpha"
	base := Credential{ID: "native-a", AccountID: "account-a", Kind: NativeCredential, User: "Phone", Identity: Identity(secret), EntryID: "entry-a", Enabled: true}
	grant := RouteGrant{
		ID: "grant-a", AccountID: "account-a", EntryID: "entry-a", EgressID: "egress-a", InboundTag: "vless-a",
		Base:  base,
		Route: Credential{ID: "route-a", AccountID: "account-a", Kind: RouteCredential, User: RouteUser("grant-a"), Identity: Identity(routeSecret), EntryID: "entry-a", EgressID: "egress-a", Enabled: true},
		Mode:  FixedMode, Enabled: true, DesiredRev: 2, AppliedRev: 1,
	}
	result, err := RenderLinks([]NativeEntry{{Material: CredentialMaterial{Credential: base, ProtocolID: secret}, Protocol: VLESSReality, Link: link}}, "account-a", FixedMode, []PublishedRoute{{Grant: grant, Protocol: VLESSReality, EntryName: "Entry-Alpha", BaseProtocolIdentity: base.Identity, RouteProtocolIdentity: grant.Route.Identity}}, false)
	if err != nil || string(result) != link+"\n" {
		t.Fatalf("unapplied route result = %q, err=%v", string(result), err)
	}
}

func TestRenderSubscriptionsRejectRoutedHysteria(t *testing.T) {
	baseSecret := "11111111-1111-4111-8111-111111111111"
	routeSecret := "22222222-2222-4222-8222-222222222222"
	base := Credential{ID: "native-a", AccountID: "account-a", Kind: NativeCredential, User: "Phone", Identity: Identity(baseSecret), EntryID: "entry", Enabled: true}
	route := Credential{ID: "route-a", AccountID: "account-a", Kind: RouteCredential, User: RouteUser("grant-a"), Identity: Identity(routeSecret), EntryID: "entry", EgressID: "egress-a", Enabled: true}
	baseMaterial := CredentialMaterial{Credential: base, ProtocolID: baseSecret, HysteriaAuth: "base-hy2", HysteriaIdentity: Identity("base-hy2")}
	routeMaterial := CredentialMaterial{Credential: route, ProtocolID: routeSecret}
	endpoint := testHysteriaEndpoint(t)
	baseLink, err := Hysteria2LinkForCredential(endpoint, baseMaterial, "🇺🇸｜Entry · HY2")
	if err != nil {
		t.Fatal(err)
	}
	grant := RouteGrant{ID: "grant-a", AccountID: "account-a", EntryID: "entry", EgressID: "egress-a", InboundTag: "meridian-entry", Base: base, Route: route, Mode: FixedMode, Enabled: true, DesiredRev: 2, AppliedRev: 2, RuntimeGood: true}
	routes := []PublishedRoute{{Grant: grant, Protocol: Hysteria2, EntryName: "｜Entry", EgressRegionCode: "TW", BaseLink: baseLink, RouteLink: baseLink, BaseProtocolIdentity: baseMaterial.HysteriaIdentity, RouteProtocolIdentity: baseMaterial.HysteriaIdentity}}
	entries := []NativeEntry{{Material: baseMaterial, Protocol: Hysteria2, Link: baseLink}}
	if _, err := RenderLinks(entries, "account-a", FixedMode, routes, false); err == nil {
		t.Fatal("routed Hysteria link subscription was accepted")
	}
	if _, err := RenderMihomo(entries, "account-a", FixedMode, routes); err == nil {
		t.Fatal("routed Hysteria Mihomo subscription was accepted")
	}
}
