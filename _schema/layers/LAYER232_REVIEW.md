# Layer 232 schema review

Source: official Telegram Desktop `dev@850fb90eb6db96c124b6da1349060f9ed00f4fe4`.
The last change to `Telegram/SourceFiles/mtproto/scheme/api.tl` is the same
commit; manifest.json locks its Git blob
`8319b7df634d145390c299404129fd161a5dbd2a` and normalized SHA-256
`76922ce265441d3f2f5188f56b69b41e1cb11354657ea547707507f8c3b93826`.
The TDesktop source path is clean and this schema commit is reachable from
`origin/dev`. No client patch is part of the schema import.

Relative to Layer 231 the schema adds 24 constructors and changes seven
constructor IDs. Canonical moves to 232; exact profiles 225–231 remain.

## Constructor changes

| Constructor | Layer 231 | Layer 232 |
|---|---|---|
| `channel` | `#d49f34c6` | `#7ad15240` (adds `in_news:flags2.21?true`, `pinned_msg_id:flags2.22?int`) |
| `channelFull` | `#a04e8d3a` | `#c4c4dd40` (adds `share_theme`) |
| `userFull` | `#6cbe645` | `#b90c2afd` (adds `wallet_card:flags2.19?wallet.Card`, `share_theme:flags2.27?ShareTheme`) |
| `messageReplyHeader` | `#1b97dd66` | `#ddb5ab75` (adds `quote_rich`) |
| `inputReplyToMessage` | `#3bd4b7c2` | `#75c1ad70` (adds `quote_rich`) |
| `wallet.nftItem` | `#3441fd1c` | `#173dfaf3` (adds `gift`) |
| `payments.getStarGiftWithdrawalUrl` | `#d06e93a8` | `#c8ca817b` (`password:InputCheckPasswordSRP` → `tmp_password:bytes`) |

New constructors: `messageActionNoPaidMessagesTokenUsed`,
`walletTransactionPeerChannel`, `walletTransactionPeerTelegramGift`,
`exportedNoPaidMessagesToken`, `shareThemeDefault`, `shareThemeEmojiBackdrop`,
`shareThemeGift`, `inputShareThemeGift`, `shareThemeBackdropOption`,
`shareThemeBackdropsNotModified`, `shareThemeBackdrops`,
`inputWalletCardSticker`, `wallet.cardSticker`, `wallet.card`,
`payments.starGiftDepositPayload`, `contacts.exportNoPaidMessagesToken`,
`contacts.importNoPaidMessagesToken`, `messages.setShareTheme`,
`messages.getShareThemeBackdrops`, `payments.withdrawStarGift`,
`payments.depositStarGift`, `payments.displayStarGift`,
`payments.deleteStarGift`, `wallet.setCardStickers`.

No update constructor or `pts`/`pts_count` field changed; the MTProto
service schema is unchanged.

## Compatibility decisions

| Scenario | Decision |
|---|---|
| `payments.getStarGiftWithdrawalUrl` in exact 232 | The request carries `tmp_password:bytes`. A non-empty temporary password is a Telegram temporary-password proof; a server must issue it through `account.getTmpPassword` and must never treat it as an SRP proof. |
| Historical `payments.getStarGiftWithdrawalUrl#d06e93a8` (225–231) | Promoted by `manifest.json` `legacy_methods` to the distinct canonical method `payments.getStarGiftWithdrawalUrlLegacy#d06e93a8` with the unchanged official fields `stargift:InputSavedStarGift password:InputCheckPasswordSRP`. The SRP proof (or `inputCheckPasswordEmpty`) reaches the server untouched; it is never dropped, never transcoded into a temporary password, and never admitted as the temporary-password method. The server must verify it as an SRP proof. Superseded 2026-10-23: v1.6.0 rejected this constructor at admission. |
| Legacy withdrawal in exact 232 | Admitted with the same wire, because the one canonical Go model owns the promoted method (the same rule as every `legacy.tl` overlay definition). The proof meaning is identical in every profile, so the method is wire-invariant and needs no policy. |
| Canonical `payments.getStarGiftWithdrawalUrl` encoded for exact 225–231 | Unavailable (generic canonical-only default): historical profiles have no temporary-password method. The legacy constructor is a different method and is never synthesized from a canonical request, so no empty SRP challenge is invented. |
| New constructors in historical profiles | Unavailable. No alias, no byte bridge. A historical client cannot call `payments.deleteStarGift`, `messages.setShareTheme`, `wallet.setCardStickers` and the other new methods. |
| `channel.in_news` / `channel.pinned_msg_id`, `channelFull.share_theme`, `userFull.wallet_card` / `userFull.share_theme`, `wallet.nftItem.gift` encoded for 225–231 | Drop (generic optional-field default). Display metadata only: no permission, consent, credential or PTS meaning, and the historical constructor keeps its own field set. |
| `messageReplyHeader.quote_rich` / `inputReplyToMessage.quote_rich` (including `draftMessage.reply_to`) encoded for 225–231 | Drop; td does not synthesize text from `RichText`. TDesktop 232 sends a rich quote with `quote_rich` only and leaves `quote_text` empty, so the drop alone would erase the quote. Server obligation: whenever the canonical value carries `quote_rich`, it must also carry the flat `quote_text` / `quote_entities` fallback (TDesktop 232 prefers `quote_rich` when both exist), which the historical projection retains. A server that cannot produce the fallback must not emit `quote_rich`. |
| Reshuffled constructor IDs | Every exact profile keeps its own wire ID. `channel`, `channelFull`, `userFull`, `messageReplyHeader`, `inputReplyToMessage` and `wallet.nftItem` are retagged per profile, never rewritten in place. |

No Layer 232 obligation needs an explicit policy entry any more. Before the
legacy promotion the 21 `payments.getStarGiftWithdrawalUrl` request
projections were explicit `reject` entries; the promotion removed those
obligations, so the entries were deleted as stale. The other new obligations
use the reviewed generic defaults listed above: a new constructor in a
historical profile is unavailable, and each new optional field is dropped.

## Verification matrix

`tlprofile/layer232_test.go` checks literal per-profile constructor IDs for the
reshuffled constructors; that the Layer 232 fields above do not change 225–231
wire bytes; the flat-quote fallback surviving the `quote_rich` drop; the literal
Layer 232 withdrawal wire and round trip; rejection of canonical withdrawals
(empty or not) for 225–231 with the output untouched; the literal legacy
withdrawal wire for SRP and empty-check proofs in every profile 225–232, its
exact decode, admission as `payments.getStarGiftWithdrawalUrlLegacy` and
handler delivery with the proof intact; and cross-profile wire-ID isolation. Keep every other exact profile wire ID
unchanged and retain the 228/229 PageBlock storage readers. Run deterministic
generation, protocol tests and `go vet`, then downstream RPC/edge tests and
command builds against the published module without a permanent local
`replace`. This protocol change allocates no PTS/QTS, durable outbox event, or
session state.

## 2026-10-22 verification

On Linux/amd64, Go 1.27.1:

- Schema import verified against the locked Git blob and normalized SHA-256;
  `canonical_layer` moved to 232 and `layer-232.tl` was written with its
  provenance.
- Policy audit: 249 retained entries, zero stale, 21 new obligations, all of
  them the `payments.getStarGiftWithdrawalUrl` projections reviewed above. The
  reviewed policy document now holds 270 entries.
- Generation is deterministic and reproduced the canonical `tg` model plus the
  `tlprofile` sidecar with `Profile232` and `ProfileCanonical = 232`.
- The locked generator budgets were re-measured for the eighth profile:
  714 projected families, 2538 rewrite variants, 957 unavailable variants,
  259 body obligations, 4 result obligations, 1899/1148/969 execution plans,
  1527 typed codec wires and 399 unprofiled wrapper invariants.

## 2026-10-23 review corrections and local verification

On darwin/arm64, Go 1.27.0, against the corrected workspace:

- Policy: the 14 historical withdrawal obligations moved from `drop`/`default`
  to `reject`; all 21 withdrawal entries now reject. The policy still holds
  270 entries with zero stale or unresolved keys.
- Two `go generate . ./gen ./tdp/internal` runs are byte-identical. The 74
  profile sidecar files occupy 13,208,005 bytes / 338,157 lines (16 MiB /
  400,000-line caps unchanged). The measured typed-wire count is 1,524 (cap
  1,530); the other locked budgets above are unchanged.
- `go test ./... -count=1`, `go vet ./...`, `CGO_ENABLED=1 go test -race
  ./tlprofile ./mtproto` and `git diff --check` pass, including the new
  `tlprofile/layer232_test.go`.
- Downstream is **not** green: with a temporary `go.work`, current `telesrv`
  fails to build at `internal/rpc/payments_star_gift_lifecycle.go:686`
  (`req.Password` no longer exists), so `internal/rpc` and
  `internal/mtprotoedge` tests cannot build either. Publishing this module is
  safe only with a separate `telesrv` business change: rewrite the withdrawal
  2FA path for `tmp_password` (refuse until `account.getTmpPassword` is
  implemented, or implement it), adapt its tests, and honor the flat-quote
  fallback obligation above. Bump `telesrv/go.mod` only after both pass.

No client UI session or deployment is part of this verification.

## 2026-10-23 legacy withdrawal promotion

Exact Layer 225–231 clients (the local TDesktop 231 fork and the TDLib 230
Android fork) send `payments.getStarGiftWithdrawalUrl#d06e93a8` with an SRP
proof. An SRP proof is a one-shot answer to a server challenge and cannot be
converted into a temporary password, so mapping the constructor onto the
canonical method is impossible without either discarding the proof or
transcoding it. Rejection kept the request safe but blocked collectible export
for every supported historical client.

The manifest now declares a reviewed `legacy_methods` entry. gotdgen checks
that every historical profile defining `#d06e93a8` carries exactly the
declared official definition, renames it to
`payments.getStarGiftWithdrawalUrlLegacy`, and adds the same definition to the
canonical model; it fails if the canonical profile stops defining the
superseding `payments.getStarGiftWithdrawalUrl`, starts defining `#d06e93a8`
itself, or no historical profile defines the wire any more. No field mapping,
adapter, runtime walker or byte bridge is involved: the historical and
canonical bodies are the same TL definition. Layer 232 keeps
`payments.getStarGiftWithdrawalUrl#c8ca817b` for `tmp_password`.

Server obligation: register both methods. The temporary-password handler
accepts only issued temporary passwords; the legacy handler accepts only an
SRP proof checked against the account's current SRP state, with the same bot,
2FA-presence, freshness, authorization and ownership gates as the
temporary-password path.

On darwin/arm64, Go 1.27.0:

- Policy audit: 249 retained, 21 stale (all withdrawal entries), zero new.
  The stale entries were removed; the policy holds 249 entries.
- `_schema/telegram.tl` was re-rendered through `RenderCanonicalSchema`; the
  only changes are the `Legacy` provenance header and the promoted definition.
- Two `go generate . ./gen ./tdp/internal` runs are byte-identical. The 74
  profile sidecar files occupy 13,198,132 bytes / 338,027 lines. Re-measured
  budgets: 714 projected families, 2538 rewrite variants, 964 unavailable
  variants (+7 for the canonical-only temporary-password method), 252 body
  obligations, 4 result obligations, 1892/1148/968 execution plans, 1522 typed
  codec wires and 400 unprofiled wrapper invariants (+1 for the legacy method).
- `go test ./... -count=1`, `go vet ./...`, `CGO_ENABLED=1 go test -race
  ./tlprofile ./mtproto` and `git diff --check` pass, including
  `gen/semantic/legacy_method_test.go` and the rewritten Layer 232 withdrawal
  tests.
- Downstream, with a temporary `go.work` over this workspace and `telesrv`
  carrying the legacy SRP handler: `go build ./...`, `go vet ./...` and
  `go test -count=1 ./...` (95 packages) pass. No client UI session or
  deployment is part of this verification.
