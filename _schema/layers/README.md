# Multi-layer TL generation

`gotdgen` treats this directory as one versioned schema universe, not as a
collection of independent generators. `manifest.json` selects the canonical
Layer and records the upstream repository, commit, blob and SHA-256 for every
input. `policy.json` contains the reviewed decisions which cannot be inferred
from TL structure alone.

The maintained exact-profile window starts at Layer 225. Layers below 225 are
intentionally absent from the manifest and fail closed at runtime; extending
the window requires importing the exact schema and reviewing its obligations.

The generated `tg` package has exactly one stable canonical Go model (the
manifest's canonical Layer). Historical schemas exist only while gotdgen builds
the transitive semantic closure. The generated `tlprofile` sidecar maps an
exact profile and wire ID to a deduplicated static execution plan. RPC results,
active updates and difference responses all use `Call.EncodeResult`,
`EncodeObject` or `FrozenObject.Encode`; there is no push-specific codec and no
fallback to canonical bytes when a projection fails.

## Synchronizing `gotd/td` upstream

Upstream is an evidence and patch source, not a branch that is merged wholesale
into the fork. Every synchronization uses this sequence:

1. Fetch upstream and record the current base, candidate new base, and exact
   commit range. Classify each non-merge commit or cohesive feature group as
   `selected` or `skipped`, with a short reason.
2. Replay selected source/runtime changes on a dedicated fork branch. Preserve
   the `github.com/iamxvbaba/td` module boundary and resolve upstream import-path
   differences in source. Keep required dependency changes in the same reviewed
   patch set as the feature that needs them.
3. Never copy, cherry-pick, or conflict-merge upstream-generated `tg/*_gen.go`
   files. They are an oracle for schema/codegen comparison only. Fork-generated
   outputs must come exclusively from the fork's manifest, semantic IR, policy,
   templates, and sparse AOT emitter.
4. Compare schema Layer, normalized SHA-256, and body before importing it. An
   identical schema with different generator/source header provenance is a
   no-op for canonical generation. A genuinely new or changed schema must use
   the import and policy-audit workflow below.
5. Run deterministic generation, targeted and full fork tests, `go vet`, source
   budgets, cross-Layer wire oracles, and downstream `telesrv` validation. Only
   after all gates pass may documentation adopt the candidate upstream base and
   a new immutable fork tag be published.

This policy applies to every future upstream update. A full upstream
merge/rebase into the release line is not a shortcut for this review.

## Adding a Layer

1. Import the exact source from a local upstream checkout. `gotdgen` discovers
   `// LAYER N`, finds the last commit on `HEAD` which changed the manifest's
   source path, verifies that commit is reachable from a fetched ref of the
   configured upstream remote and verifies the bytes against that commit's Git
   object, computes SHA-256, writes `layer-N.tl`,
   updates manifest provenance/canonical selection, and rebuilds
   `_schema/telegram.tl` with the locked overlays. No network is used:

   The import stages and syncs the complete output set before replacing any
   tracked file. Layer and canonical schema files are replaced first;
   `manifest.json` is replaced last as the commit marker. A replacement error
   rolls every target back to the complete previous set.

   ```powershell
   go run ./cmd/gotdgen --schema-manifest _schema/layers/manifest.json `
     --schema-import ..\tdesktop\tdesktop\Telegram\SourceFiles\mtproto\scheme\api.tl
   ```

   Fetch the upstream checkout before importing. A copied source must still be
   verified against a local checkout; pass
   `--schema-import-git-dir <checkout>` and, when needed, the exact
   `--schema-import-commit <40-hex>`. Replacing different bytes/provenance for
   an already-recorded Layer additionally requires `--schema-import-replace`.
2. Audit the existing policy without weakening an existing decision. Unlike
   normal generation, audit mode reports stale keys instead of aborting before
   a review artifact can be produced:

   ```powershell
   go run ./cmd/gotdgen --schema-manifest _schema/layers/manifest.json `
     --layer-policy _schema/layers/policy.json `
     --layer-policy-audit _schema/layers/policy.audit.json `
     --layer-policy-merge _schema/layers/policy.next.json
   ```

   The audit has three deterministic sections: `retained` (exact key still
   valid), `stale` (never copied forward), and `new` (empty-action skeleton).
   The merged file contains retained plus new entries and deliberately fails
   normal generation until every new action is reviewed.
3. Review every new obligation. Mechanical ID/body reuse needs no policy.
   Removed fields, replacements, old-only definitions, changed RPC results and
   unavailable update constructors require an explicit `reject`, `drop`,
   `default`, `alias`, `project` or typed `adapter` decision. Copy only reviewed
   entries into `policy.json`; stale keys and unresolved obligations are hard
   generation errors.
4. Regenerate and verify:

   ```powershell
   go generate .
   go test ./gen ./cmd/gotdgen ./tg ./tlprofile
   go vet ./gen ./cmd/gotdgen
   git diff --check
   ```

Adding an unchanged 228/229 profile therefore only extends generated profile
metadata and switch cases. A genuinely changed schema stops at generation time
until its semantic policy and typed hook are supplied.

## Client-private RPC overlays

`client_rpc_overlays` in `manifest.json` is intentionally separate from the
official Layer universe. Each entry locks a derived TL file by SHA-256 and the
upstream client commit/source blobs from which its serializers were audited.
gotdgen binds every private function to the current canonical function by
semantic name and emits a direct CRC switch plus static typed field decoders.
Nested constructors always decode with the connection's exact Layer profile.

Schema-identical fields map automatically. Renames, type converters and
explicitly consumed/dropped client-only fields are declared per method in the
manifest. Missing optional fields and required primitive/vector zero values can
be generated mechanically; a missing required object, result mismatch,
unmapped source field, unknown converter, CRC collision or stale rule aborts
generation. Upgrading canonical Layer therefore needs only the ordinary Layer
import and regeneration when the comparison remains mechanical; otherwise the
generator stops for an explicit review instead of guessing.

The current DrKLO inputs keep the 15 long-lived core private RPCs and four
theme RPCs in separate overlays. They generate into the `tlprofile` sidecar.
Production adapters call `UnknownMethodView.AdaptClientRPCOverlay`, which
shares the outer request decode budget and authoritative canonical admission
bridge. Overlay IDs are classified per exact profile, so a private ID that
collides with a historical official constructor is never globally stolen.
There is no runtime TL parser, schema map, reflection walker, or client-driven
Layer selection in this path.

## Runtime invariants

- `Dispatcher.Admit` preserves a caller-frozen exact profile and rejects a
  conflicting inner `invokeWithLayer`. `Dispatcher.AdmitDefault` instead
  treats the caller profile as inherited fallback state for naked RPCs: the first `invokeWithLayer`
  anywhere in the transparent wrapper chain may replace it, after which
  repeated selectors must agree. The admitted request reports the effective
  profile separately from explicit profile evidence.
- With neither a frozen nor inherited profile, the dispatcher accepts only
  `invokeWithLayer` or a terminal RPC whose request graph and complete result
  `TypeRef` graph were proven at generation time to be wire-identical in every
  loaded profile. Such an invariant terminal (for example
  `auth.bindTempAuthKey`) publishes neither an effective profile nor profile
  evidence; its canonical codec route remains an internal representative.
- Request admission consumes the complete outer wire value and freezes the
  exact result `TypeRef`, wire ID and request digest.
- Wrapper metadata is preserved outer-to-inner. Wrappers with session,
  ordering or update-suppression meaning must be consumed explicitly.
- If at least one generated wrapper is completely decoded but its innermost
  constructor misses the exact-profile RPC switch, admission may return
  `UnknownTerminalError`. It carries the exact profile, terminal wire
  ID and complete remaining terminal size plus a private immutable
  outer-to-inner wrapper identity chain. Naked unknown constructors and
  malformed wrappers never receive this proof. The proof is not admission:
  an edge consumer must whitelist both a closed terminal schema and every
  wrapper semantic before handling a non-API service terminal; otherwise it
  must fail closed. No decoder cursor or runtime TL walker is exposed.
- Flags are rebuilt by `(flags word, bit)` groups. Present-empty slices remain
  present; every value-bearing member of a set bit is encoded atomically.
- Encoding is transactional. Rejected adapters, partial projections, malformed
  flags and size limits leave the caller's buffer unchanged.
- Decode depth and vector lengths are bounded. TL strings and byte strings must
  fit the protocol's 24-bit length header.
- Frozen values may be prepared once per exact profile/call identity. Wire bytes
  are never reused across unequal profiles or result `TypeRef` identities.
- Proactive updates still require a real frozen target profile. Only a result
  bound to a generated wire-invariant RPC may be encoded before profile
  evidence exists; the exception cannot be used as a generic unknown-profile
  fallback.

The source emitter coalesces identical body, preflight and result execution
plans. Dispatch still has an exact switch for every `(profile, wire ID)` route,
but byte-identical routes share generated helpers; identical wrapper probes
share one probe helper as well. The current source gate rejects any regenerated
`tg/tl_layer*_gen.go` and caps the complete generated `tlprofile` sidecar at
16 MiB / 400k lines. Adding an unchanged future Layer therefore grows route
metadata instead of cloning the full TL catalog. None of these paths uses
reflection, a runtime schema walker or a dynamic schema map.
