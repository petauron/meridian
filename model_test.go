package meridian

import (
	"strings"
	"testing"
)

func TestRouteGrantRequiresAppliedHealthyIndependentCredential(t *testing.T) {
	baseSecret := "11111111-1111-4111-8111-111111111111"
	routeSecret := "22222222-2222-4222-8222-222222222222"
	grant := RouteGrant{
		ID: "grant-a", AccountID: "account-a", EntryID: "entry-a", EgressID: "egress-a", InboundTag: "vless-a",
		Base:  Credential{ID: "credential-base", AccountID: "account-a", Kind: NativeCredential, User: "Phone", Identity: Identity(baseSecret), EntryID: "entry-a", Enabled: true},
		Route: Credential{ID: "credential-route", AccountID: "account-a", Kind: RouteCredential, User: RouteUser("grant-a"), Identity: Identity(routeSecret), EntryID: "entry-a", EgressID: "egress-a", Enabled: true},
		Mode:  FixedMode, Enabled: true, DesiredRev: 3, AppliedRev: 3, RuntimeGood: true,
	}
	if err := grant.Validate(); err != nil || !grant.Publishable() {
		t.Fatalf("valid grant = %#v, err=%v", grant, err)
	}
	grant.RuntimeGood = false
	if grant.Publishable() {
		t.Fatal("runtime-blocked route was publishable")
	}
	grant.RuntimeGood = true
	grant.Route.Identity = grant.Base.Identity
	if err := grant.Validate(); err == nil {
		t.Fatal("shared native/route identity was accepted")
	}
}

func TestPlanImportRejectsDuplicatedSubscriptionAuthority(t *testing.T) {
	account := Account{ID: "account-a", DisplayName: "Phone", Plan: AccountPlan{Enabled: true}, DesiredRevision: 1}
	secret := "11111111-1111-4111-8111-111111111111"
	credential := Credential{ID: "native-a", AccountID: account.ID, Kind: NativeCredential, User: "Phone", Identity: Identity(secret), EntryID: "entry-a", Enabled: true}
	item := ImportAccount{Account: account, SubscriptionTokenFingerprint: SubscriptionTokenFingerprint("token-a"), Credentials: []Credential{credential}, Usage: []UsageMember{{CredentialID: credential.ID, Active: true}}}
	plan, err := PlanImport([]ImportAccount{item})
	if err != nil || len(plan.Accounts) != 1 || len(plan.SHA256) != 64 {
		t.Fatalf("import plan = %#v, err=%v", plan, err)
	}
	duplicate := item
	duplicate.Account.ID = "account-b"
	duplicate.Account.DisplayName = "Router"
	duplicate.Credentials = []Credential{{ID: "native-b", AccountID: "account-b", Kind: NativeCredential, User: "Router", Identity: Identity(secret), EntryID: "entry-b", Enabled: true}}
	duplicate.Usage = []UsageMember{{CredentialID: "native-b", Active: true}}
	if _, err := PlanImport([]ImportAccount{item, duplicate}); err == nil || !strings.Contains(err.Error(), "credential") {
		t.Fatalf("duplicated identity err=%v", err)
	}
}

func TestProjectQuotaSharesOneAccountPlan(t *testing.T) {
	projection, err := ProjectQuota(AccountPlan{TotalBytes: 100, Enabled: true}, []UsageMember{
		{CredentialID: "native", Observed: 40, Active: true},
		{CredentialID: "route-a", Observed: 30, Active: true},
		{CredentialID: "route-b", Observed: 20, Active: true},
	})
	if err != nil || projection.UsedBytes != 90 || !projection.Enabled || len(projection.Gates) != 3 {
		t.Fatalf("projection = %#v, err=%v", projection, err)
	}
	projection, err = ProjectQuota(AccountPlan{TotalBytes: 100, Enabled: true}, []UsageMember{
		{CredentialID: "native", Observed: 50, Active: true},
		{CredentialID: "route-a", Observed: 30, Active: true},
		{CredentialID: "route-b", Observed: 20, Active: true},
	})
	if err != nil || projection.Enabled {
		t.Fatalf("exhausted projection = %#v, err=%v", projection, err)
	}
}
