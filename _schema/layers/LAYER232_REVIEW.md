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
| `payments.getStarGiftWithdrawalUrl` in exact 232 | The request carries `tmp_password:bytes`. A non-empty temporary password is a Telegram temporary-password proof this fork does not issue, so a server without `account.getTmpPassword` must refuse it rather than treat it as a 2FA proof. |
| Historical `payments.getStarGiftWithdrawalUrl` (225–231) admitted to canonical 232 | Reject at admission, for both `inputCheckPasswordSRP` and `inputCheckPasswordEmpty`. The SRP proof has no canonical counterpart; dropping it would admit an authenticated financial request as "no temporary password", which the server could not distinguish from a request that carried no proof. It is never promoted into a temporary password. Same precedent as the removed Layer 230 Firebase credential. Historical clients therefore cannot withdraw until they upgrade to a Layer 232 client using `account.getTmpPassword`. |
| Canonical `payments.getStarGiftWithdrawalUrl` encoded for exact 225–231 | Reject every request, including an empty `tmp_password`: the legacy constructor requires an SRP challenge the canonical value cannot supply, and inventing an empty challenge would downgrade an authenticated request to an unauthenticated one. |
| New constructors in historical profiles | Unavailable. No alias, no byte bridge. A historical client cannot call `payments.deleteStarGift`, `messages.setShareTheme`, `wallet.setCardStickers` and the other new methods. |
| `channel.in_news` / `channel.pinned_msg_id`, `channelFull.share_theme`, `userFull.wallet_card` / `userFull.share_theme`, `wallet.nftItem.gift` encoded for 225–231 | Drop (generic optional-field default). Display metadata only: no permission, consent, credential or PTS meaning, and the historical constructor keeps its own field set. |
| `messageReplyHeader.quote_rich` / `inputReplyToMessage.quote_rich` (including `draftMessage.reply_to`) encoded for 225–231 | Drop; td does not synthesize text from `RichText`. TDesktop 232 sends a rich quote with `quote_rich` only and leaves `quote_text` empty, so the drop alone would erase the quote. Server obligation: whenever the canonical value carries `quote_rich`, it must also carry the flat `quote_text` / `quote_entities` fallback (TDesktop 232 prefers `quote_rich` when both exist), which the historical projection retains. A server that cannot produce the fallback must not emit `quote_rich`. |
| Reshuffled constructor IDs | Every exact profile keeps its own wire ID. `channel`, `channelFull`, `userFull`, `messageReplyHeader`, `inputReplyToMessage` and `wallet.nftItem` are retagged per profile, never rewritten in place. |

Only the 21 `payments.getStarGiftWithdrawalUrl` request projections (all
`reject`) are explicit policy entries. The other new obligations use the
reviewed generic defaults listed above: a new constructor in a historical
profile is unavailable, and each new optional field is dropped.

## Verification matrix

`tlprofile/layer232_test.go` checks literal per-profile constructor IDs for the
reshuffled constructors; that the Layer 232 fields above do not change 225–231
wire bytes; the flat-quote fallback surviving the `quote_rich` drop; the literal
Layer 232 withdrawal wire and round trip; rejection, without consuming input,
of historical SRP and empty-check withdrawals in decode and admission; rejection
of canonical withdrawals (empty or not) for 225–231 with the output untouched;
and cross-profile wire-ID isolation. Keep every other exact profile wire ID
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
