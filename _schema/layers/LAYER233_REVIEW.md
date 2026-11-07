# Layer 233 schema review

Source: official Telegram Desktop `ready@a6df181ac41763a9ff9edae264dad407fb1c35ec`
(the stable/`ready` branch; `dev@93624f698` carries byte-identical
`api.tl`). `manifest.json` locks Git blob
`4e29f2deb45a083196a8c7ac09cdedf2600c1a10` and normalized SHA-256
`c2bbefcaec3285ba46c8af8f429c7012fe5aea253fd16d4be089f91c7c3fa71e`. The
TDesktop source path is clean and this schema commit is reachable from
`origin/ready`. No client patch is part of the schema import.

Relative to Layer 232 the schema adds 27 definitions (19 constructors and 8
functions) and changes three: `user` and `channel` gain flags under unchanged
IDs, and `channels.toggleForum` gains a required field and a new ID. Canonical
moves to 233; exact profiles 225–232 remain generated.

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
`pts`/`pts_count` field changes; the MTProto service schema is unchanged. One
new update, `updateMessageWidgetData`, carries `pts`/`pts_count` (see its
decision below).

## Compatibility decisions

| Scenario | Decision |
|---|---|
| `channels.toggleForum` in 225–232 (profile→canonical) | Default `sidebar` to `false`. `sidebar` is per-channel layout metadata (`channel.forum_sidebar`). An old client only knows the list and tabs layouts, and the `tabs` flag it sends still chooses between them, so `sidebar=false` is exactly the layout it asked for. Every client in use at import time sends `#3ff75734` (Telegram Desktop 7.3.1 and Telegram Android 13.0.3 at Layer 231, TDLib 1.8.69 at Layer 232); rejecting it would leave all of them unable to turn forums on or off, as the v1.6.0 `getStarGiftWithdrawalUrl` rejection left historical clients without gift export until v1.7.0 promoted the legacy method. Required profile→canonical precedent: `keyboardButton.type` (typed default adapter) and `auth.sentCodePaymentRequired.premium_days` (`default`). |
| Canonical `channels.toggleForum` encoded for 225–232 (canonical→profile) | The required-field projection mechanically drops `sidebar` and emits the historical `#3ff75734` body. Server obligation: a canonical `#a2952b54` request carries the client's explicit `sidebar`; a historical request arrives with `sidebar=false`. A server that wants to keep a sidebar layout chosen by a Layer 233 client may preserve it for historical sessions. |
| `channel.forum_sidebar`, `user.bot_forum_sidebar` encoded for 225–232 | Drop (generic optional-field default). Layout metadata only: no permission, consent, credential or PTS meaning, and the historical constructor keeps its field set. Wire IDs are unchanged, so the flags are canonical-only for 225–232. |
| Six new updates (`updateEditForumTopic`, `updateBotWidgetQuery`, `updateInlineBotWidgetQuery`, `updateBotWidgetCopyQuery`, `updateMessageWidgetData`, `updateTranscribedAudioFile`) projected to 225–232 | Drop. Every prior non-wallet update added in Layers 225–229 was dropped (`updateJoinChatWebViewDecision`, `updateNewBotConnection`, `updateWebBrowserException`, `updateWebBrowserSettings`, the four ephemeral updates, and `updateEphemeralBotCallbackQuery`); only the Layer 230 wallet/TON-session updates were rejected. The widget, transcription and forum-UI features are unavailable to old profiles, and a dropped interactive widget query cannot authorize a wallet transaction. `updateMessageWidgetData` is the first dropped update with `pts`/`pts_count`: a 225–232 session sees a pts gap when it is emitted and recovers through `updates.getDifference`/`updates.getChannelDifference`, where the update is dropped again from `other_updates` while the returned state still covers its pts. A server emitting it must account for that gap. |
| New widget methods, results and `WidgetQueryType`/`UserWallet` types in historical profiles | Unavailable (generic canonical-only default). No alias and no byte bridge. `messageMediaWidget` in the optional `message.media` is projected to `messageMediaUnsupported#9f84f49e` for 225–232 (new-only project hook `layerProjectLegacyWidgetMedia`): a widget-only message reaches an older client as an explicit unsupported-media placeholder instead of an empty text message. |
| Legacy `payments.getStarGiftWithdrawalUrlLegacy#d06e93a8` | Still owned by the canonical Go model. Exact profile 232 defines the temporary-password method `#c8ca817b`, not `#d06e93a8`, so unlike the Layer 232 record it no longer admits the promoted legacy wire; the SRP constructor remains available in exact profiles 225–231. |

The policy has 313 entries: the 249 reviewed decisions are retained unchanged,
and 64 new obligations are explicit: eight `channels.toggleForum` request
defaults, 48 update drops (six constructors × eight historical profiles) and
eight `messageMediaWidget` new-only projections, one per historical profile.
The eight projections share one typed hook (`layerProjectLegacyWidgetMedia`);
no other new obligation uses a hook, and no stale key remains.

## Verification matrix

`tlprofile/layer233_test.go` checks the canonical profile pair; the unchanged
`channel`/`user` wire IDs with literal canonical IDs and non-leakage of the two
new layout flags into 225–232; the literal `#a2952b54` canonical
`channels.toggleForum` round trip versus the dropped `#3ff75734` historical
encoding, with historical admission as `sidebar=false`; and, for all six new
updates, canonical round trips, bare-encode rejection and drops inside an
`updates` container that keep the neighbouring update.
`tlprofile/layer233_widget_projection_test.go` round-trips a `message` with
`messageMediaWidget` media through every exact profile: 225–232 decode
`messageMediaUnsupported` and 233 keeps the widget. Keep every prior exact
profile wire ID and the existing Layer 230/231/232 oracles. Run deterministic
generation, protocol tests and `go vet`, then downstream RPC/edge tests and
command builds against the published module without a permanent local
`replace`. This protocol change allocates no PTS/QTS, durable outbox event, or
session state.

## Local verification

On Windows/amd64, Go 1.27.3:

- Schema import verified against the locked Git blob and normalized SHA-256;
  `canonical_layer` moved to 233 and `layer-233.tl` was written with its
  provenance.
- Policy audit: 249 retained entries, zero stale, 56 new obligations, all
  reviewed above. The reviewed policy document then held 305 entries; the
  widget media projection below took it to 313.
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

Maintainer re-verification on macOS/arm64, Go 1.27.0, from an LF checkout:

- The Windows run had checked out `gen/_template` with CRLF. `go/scanner`
  strips `\r` from line comments, so `go/printer` skipped doc-comment
  reformatting and 1121 `tg`, `tg/e2e` and `mt` files differed from an LF
  generation in comment layout only. `.gitattributes` now pins `*.tmpl` to LF,
  and the files are regenerated.
- The `channels.toggleForum` requests switched from reject to default; the
  execution-plan audit and `tlprofile` were regenerated, and a second
  `go generate .` run is byte-identical.
- `go build ./...`, the `go vet` set above and `go test ./... -count=1` pass.
  A downstream server build against this tree through a temporary `go.work`
  passes its edge tests.

Widget media projection follow-up (pull request #6), maintainer verification on
macOS/arm64, Go 1.27.0, from an LF checkout:

- `messageMediaWidget` resolves its eight new-only obligations with
  `LayerResolveProject` and the hook `layerProjectLegacyWidgetMedia` instead of
  the default drop, which had left a widget-only message empty on 225–232. The
  policy grows from 305 to 313 entries.
- It is the first family-level project hook in the sparse codec. The family
  project function calls the hook, and the `MessageMedia` class encoder
  dispatches on the projected constructor. The family body keeps the drop for
  225–232 instead of calling the generic object encoder, which the sparse codec
  never generates. Only the `MessageMediaWidget` family changes in the
  regenerated `tlprofile`.
- The four stale `telegram/` convenience files (entity and styling options,
  typing actions and cached queries) are regenerated, so `make check_generated`
  is clean again. The typing builders `TextDraft`, `RichMessageDraft` and
  `InputSendMessageRichMessageDraft` gain the schema's `canStop` and
  `keepOnStop` arguments.
- A full `go generate ./...` run is byte-identical. `go build ./...`,
  `go vet ./...` and `go test ./... -count=1` (75 packages) pass. A downstream
  server build against this tree through a temporary `go.work` passes its edge
  tests and its Layer 232/233 RPC tests.

No client UI session or deployment is part of this verification.
