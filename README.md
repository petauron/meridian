# Petauron Meridian

Meridian is Petauron's access and subscription domain for managed proxy
entries. It owns account plans, stable access credentials, route grants,
shared usage accounting, and deterministic subscription rendering.

Meridian is not a general-purpose proxy panel and does not fork 3x-ui. Vastora
remains the desired-state control plane and its Agent remains responsible for
validating and atomically applying Xray configurations on managed nodes.
Meridian supplies the authoritative, runtime-independent domain rules that
both sides share.

## Product boundary

- **Vastora** deploys applications, selects nodes, authorizes operations, and
  records desired/applied revisions.
- **Meridian** defines accounts, credentials, route grants, shared quotas, and
  subscription output.
- **Vastora Agent** applies the resulting Xray identities and routes, reports
  monotonic counters, and confirms runtime health.
- **Pulse** observes infrastructure health.

There is one writer for each credential. A migration may import an existing
3x-ui identity once, but released runtime code must not retain dual reads,
dual writes, aliases, or a 3x-ui fallback after cutover.

## Initial scope

The first vertical slice supports managed VLESS/REALITY entries, fixed
entry-to-egress route credentials, conventional base64 subscriptions, Mihomo
YAML, and one shared account quota across all credentials. The APIs are pure
and deterministic so callers can persist state in their own forward-only
schema and compare desired output before applying a runtime revision.

## Security model

Persisted domain records refer to credential fingerprints and caller-owned
encrypted secrets. Xray rendering necessarily receives the decrypted UUID and
REALITY private key for the duration of one in-process projection; these
secret-bearing values must stay in encrypted task/state envelopes and are
never included in errors, receipts, or event text. Raw VLESS links are accepted
only as bounded renderer input and are never suitable for logs. Subscription
rendering rejects cross-account routes, duplicate identities, ambiguous names,
unsupported transports, and mismatched REALITY material.

## Core API

- `PlanImport` validates and fingerprints a complete one-time authority
  transfer before callers mutate durable state.
- `ProjectQuota` applies one shared account allowance across native and routed
  credentials without splitting or duplicating the allowance.
- `RenderLinks` and `RenderMihomo` produce deterministic subscriptions from
  applied, healthy route receipts.
- `BuildDesiredArtifact` creates a complete Xray candidate and binds it to a
  revision and SHA-256 digest.
- `VerifyAppliedReceipt` prevents a stale or different runtime from being
  reported as the desired state.
- `ParseXrayUserCounters` maps exact Xray user counters back to credential IDs
  without relying on delimiter splitting.

## Status

The domain module is being integrated into Vastora as the replacement for the
remaining 3x-ui controller adapter. It is not yet a released standalone
service.
