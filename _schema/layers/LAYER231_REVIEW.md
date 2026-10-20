# Layer 231 schema review

Source: Telegram Desktop `dev`, official schema commit
`91ed9dbf2004ed5531a94ecc4fe5751e008dc8ed` ("Update API scheme to layer 231.",
2026-10-17), reachable from fetched upstream/dev. The exact blob
(`01e31d552dd496e592f262989c8452f7a3ce411a`) and SHA-256
(`0921a7bba8be9ad7e615d1d4f8e068126a421e3cba18f499a16765f1ab6129bf`) are
locked in manifest.json. No client patch is part of this change. The gotd
upstream code baseline is unchanged.

The canonical profile moves from 230 to 231. Profiles 225–230, including the
228/229 PageBlock storage profiles, remain generated. Schema is imported from
Git-verified source; tg and the sparse exact-profile sidecar are regenerated
through the existing semantic IR, never copied from an upstream generator.

## Reviewed conversion decisions

The policy audit retained 243 exact decisions, reported no stale keys, and found
**no new obligations**. The Layer 231 schema delta against Layer 230 is
mechanical:

| Change | Decision |
|---|---|
| `urlAuthResultRequest` CRC `#3cd623ec` → `#bdca6167` | The older CRC stays exact for profiles 225–230. The new constructor adds `request_wallet:flags.8?true`, `tc_session_id:flags.9?long`, `tc_client_id:flags.10?string`; historical profiles cannot express them and keep the previous constructor shape. |
| `messages.acceptUrlAuth` CRC `#67a3f0de` → `#18dd4e92` | The older CRC stays exact for profiles 225–230. The new constructor adds `tonconnect_session:flags.5?InputTonConnectOauthSession`; historical profiles keep the previous constructor shape. |
| `inputTonConnectOauthSession#629f66ce` | New type in the canonical profile only. It carries `session_id`, `challenge_answer`, `body` and `proof:WalletOwnershipProof`; it is referenced solely by the Layer 231 `messages.acceptUrlAuth` result. |

No field was removed, no constructor was renamed, and no historical profile
gained or lost a family. The three items above are pure additions on top of
the Layer 230 wire, so no new policy entry was required and no existing
decision was weakened.

## Verification matrix

Cover exact 225–231 request/result/object encoding, same-CRC flags, legacy
string adapters in both directions, optional presence, rich-entity rejection,
new update rejection on old profiles, removed credential rejection, nested
wallet result classes, malformed input and transactional cursor/buffer state.
Retain PageBlock 228/229 decoding, old keyboard adapters, client RPC overlays,
wrapper admission and frozen-result tests. Run deterministic generation,
generator/source-budget gates, full root module tests/vet, targeted races and
downstream telesrv compile/codec tests using a temporary go.work (no go.mod
replace).

Locked generator budgets were re-counted for the added profile: projected
families 687→689, rewrite variants 2101→2107, unavailable variants 783→789,
body obligation variants 212→224, field-projection obligations 181→205,
new-only obligations 783→789, sparse typed codec wire budget 1500→1520 (the
observed count is 1501). The profile catalog assertion now covers 225–231.
The semantic universe lock follows the same window: signature changes
56→58 and the Layer 225 signature baseline 48→50.
