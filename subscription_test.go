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
	entries := []NativeEntry{{Credential: grant.Base, Link: baseLink}}
	result, err := RenderLinks(entries, "account-a", FixedMode, []PublishedRoute{{Grant: grant, EntryName: "｜Entry-Alpha", EgressRegionCode: "TW", BaseLink: baseLink, RouteLink: routeLink}}, true)
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
	result, err := RenderLinks([]NativeEntry{{Credential: credential, Link: link}}, "account-a", FixedMode, nil, false)
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
	result, err := RenderLinks([]NativeEntry{{Credential: base, Link: link}}, "account-a", FixedMode, []PublishedRoute{{Grant: grant, EntryName: "Entry-Alpha"}}, false)
	if err != nil || string(result) != link+"\n" {
		t.Fatalf("unapplied route result = %q, err=%v", string(result), err)
	}
}
