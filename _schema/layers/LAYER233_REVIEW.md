# Layer 233 schema review

Source: official Telegram Desktop `ready@a6df181ac41763a9ff9edae264dad407fb1c35ec`
(the stable/`ready` branch; `dev@93624f698` carries byte-identical
`api.tl`). `manifest.json` locks Git blob
`4e29f2deb45a083196a8c7ac09cdedf2600c1a10` and normalized SHA-256
`c2bbefcaec3285ba46c8af8f429c7012fe5aea253fd16d4be089f91c7c3fa71e`. The
TDesktop source path is clean and this schema commit is reachable from
`origin/ready`. No client patch is part of the schema import.

Relative to Layer 232 the schema adds 27 constructors, three new functions
groups and changes two wire-visible definitions. Canonical moves to 233; exact
profiles 225–232 remain generated.

## Constructor changes

| Constructor | Layer 232 | Layer 233 |
|---|---|---|
| `user` | `#b1b8cc83` | `#b1b8cc83` (adds `bot_forum_sidebar:flags2.22?true`) |
| `channel` | `#7ad15240` | `#7ad15240` (adds `forum_sidebar:flags2.23?true`) |
| `channels.toggleForum` | `#3ff75734` | `#a2952b54` (adds required `sidebar:Bool`) |

New constructors: `inputMediaWidget#44f6c04e`, `inputMediaWidgetId#7c855cb5`,
`messageMediaWidget#2238f214`, `inputBotInlineMessageWidget#f597a3f0`,
`inputBotInlineMessageWidgetId#fe8b792b`, `botInlineMessageWidget#baa17834`,
`userWallet#454d2737`, `widgetQueryTypeData#16d8cf0b`,
`widgetQueryTypeSignData#192047dc`, `widgetQueryTypeSentTransaction#ff3b33b4`,
`messages.botWidgetQueryAnswer#95b73023`,
`messages.widgetPublicDataNotModified#b9196e57`, `messages.widgetPublicData#4493bff`,
`updateEditForumTopic#2366665c`, `updateBotWidgetQuery#b373fe68`,
`updateInlineBotWidgetQuery#a5f07e08`, `updateBotWidgetCopyQuery#fa89bef7`,
`updateMessageWidgetData#bbb078dd`, `updateTranscribedAudioFile#734c2e7b`.

New functions: `messages.getBotWidgetQueryAnswer#5c550579`,
`messages.setBotWidgetQueryAnswer#f8cdec6a`,
`messages.editMessageWidgetData#f624bc1e`,
`messages.editInlineBotMessageWidgetData#6396ce2b`,
`messages.editWidgetData#eac343c0`, `messages.getWidgetPublicData#6fc53181`,
`messages.transcribeAudioFile#72d2ef7`, `messages.getTranscriptionLangs#ba6cd9aa`.

The six new `Update` constructors are added, but no existing update's
`pts`/`pts_count` field changes; the MTProto service schema is unchanged.

## Compatibility decisions

| Scenario | Decision |
|---|---|
| `channels.toggleForum` in 225–232 (profile→canonical) | Reject admission. A historical request cannot express the required `sidebar` layout decision; defaulting it to `false` would let an old client silently overwrite the account's sidebar preference. This matches the only prior request-required-field decisions (`inputStorePaymentAuthCode.premium_days`, `payments.getStarGiftWithdrawalUrl.tmp_password`), which reject rather than invent a value. |
| Canonical `channels.toggleForum` encoded for 225–232 (canonical→profile) | The required-field projection mechanically drops `sidebar` and emits the historical `#3ff75734` body. Server obligation: a request admitted as the canonical `#a2952b54` must treat `sidebar` as the client's explicit choice; a rejected historical body must never be re-interpreted as a sidebar change. |
| `channel.forum_sidebar`, `user.bot_forum_sidebar` encoded for 225–232 | Drop (generic optional-field default). Layout metadata only: no permission, consent, credential or PTS meaning, and the historical constructor keeps its field set. Wire IDs are unchanged, so the flags are canonical-only for 225–232. |
| Six new updates (`updateEditForumTopic`, `updateBotWidgetQuery`, `updateInlineBotWidgetQuery`, `updateBotWidgetCopyQuery`, `updateMessageWidgetData`, `updateTranscribedAudioFile`) projected to 225–232 | Drop. Every prior non-wallet update added in Layers 225–229 was dropped (`updateJoinChatWebViewDecision`, `updateNewBotConnection`, `updateWebBrowserException`, `updateWebBrowserSettings`, the four ephemeral updates, and `updateEphemeralBotCallbackQuery`); only the Layer 230 wallet/TON-session updates were rejected. The widget, transcription and forum-UI features are unavailable to old profiles, and a dropped interactive widget query cannot authorize a wallet transaction. |
| New widget methods, results and `WidgetQueryType`/`UserWallet` types in historical profiles | Unavailable (generic canonical-only default). No alias and no byte bridge. |
| Legacy `payments.getStarGiftWithdrawalUrlLegacy#d06e93a8` | Still owned by the canonical Go model. Exact profile 232 defines the temporary-password method `#c8ca817b`, not `#d06e93a8`, so unlike the Layer 232 record it no longer admits the promoted legacy wire; the SRP constructor remains available in exact profiles 225–231. |

The policy has 305 entries: the 249 reviewed decisions are retained unchanged,
and 56 new obligations are explicit (eight `channels.toggleForum` request
rejections and 48 update drops, six constructors × eight historical profiles).
No obligation uses a hook; no stale key remains.

## Verification matrix

`tlprofile/layer233_test.go` checks the canonical profile pair; the unchanged
`channel`/`user` wire IDs with literal canonical IDs and non-leakage of the two
new layout flags into 225–232; the literal `#a2952b54` canonical
`channels.toggleForum` round trip versus the dropped `#3ff75734` historical
encoding, with rejection and an untouched cursor when a historical request is
admitted; and canonical round trips plus historical rejection for all six new
updates. Keep every prior exact profile wire ID and the existing Layer 230/231/232
oracles. Run deterministic generation, protocol tests and `go vet`, then
downstream RPC/edge tests and command builds against the published module
without a permanent local `replace`. This protocol change allocates no PTS/QTS,
durable outbox event, or session state.

## Local verification

On Windows/amd64, Go 1.27.3:

- Schema import verified against the locked Git blob and normalized SHA-256;
  `canonical_layer` moved to 233 and `layer-233.tl` was written with its
  provenance.
- Policy audit: 249 retained entries, zero stale, 56 new obligations, all
  reviewed above. The reviewed policy document now holds 305 entries.
- `go generate .` reproduced the canonical `tg` model plus the `tlprofile`
  sidecar with `Profile233` and `ProfileCanonical = 233`; the execution-plan
  audit was regenerated.
- Re-measured locked budgets for the ninth profile: 743 projected families,
  2888 rewrite variants, 1181 unavailable variants, 266 body obligations,
  4 result obligations, 2172/1193/1102 execution plans, 1544 typed codec wires
  and 399 unprofiled wrapper invariants.
- `go test ./... -count=1`, `go build ./...` and `go vet ./gen ./gen/semantic
  ./cmd/gotdgen ./tlprofile ./mtproto ./tg` pass, including the new
  `tlprofile/layer233_test.go`.

No client UI session or deployment is part of this verification.
