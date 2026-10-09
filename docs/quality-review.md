# Cyclo quality review — 2026-10-09

Baseline: `f8d4549` (latest `main` when this review starts).
Initial tool: installed local Cyclo, revision `7421c3d1f15dc8e65d9f0453d7b206a470ec68ef`, built 2026-10-09.
The installed binary is newer than the local source checkout. This review uses the installed binary and its `--skill` instructions.

## Rerun after the Cyclo typed-nil fix

The active rerun uses installed Cyclo `5890ecc9abedde8fd3e95aa87c3d28ef90c6fd91` (GitHub PR #34 merged). The binary is built from clean GitHub master. Both current linter main (`f8d4549`) and this PR (`841648e`, before this report update) are scanned with the same tool. Main has no newer commits.

**All 13 typed-nil false alarms are gone on both main and this PR.** The valid nil checks remain unchanged. Full results:

| Scan | Current main | This PR |
| --- | ---: | ---: |
| Production quality findings, current defaults | 67 | 67 |
| Pattern candidates | 73 | 74 |
| Typed-nil candidates | 0 | 0 |

The review uses the current **200-line function-length limit**. The active production scan reports **67 findings** and no function-length findings. The earlier 72/70 counts below are historical results from the initial tool; they are not the current policy.

The 74 current pattern candidates are 39 `thin_slice`, 19 `chop`, 8 `barrier_slice`, 3 `ccgraph_clone`, 2 `forcetypeassert`, and one each of `value_object`, `primitive_obsession`, and `parameterize`. The 66 slicing candidates are new dependency-analysis signals; they are not 66 confirmed bugs. The remaining eight candidates retain the review decisions below. The dry-run fixer reports zero automatic edits; an apply run also leaves the tree unchanged. The current score is 0/100 with 74 active candidates; no warning is suppressed to change that score.

The scan including tests reports 159 quality findings. The changed-function scan against `f8d4549` reports 11: seven effect-density findings, two mutated-target findings, one mutation-per-target finding, and one parameter-count finding. Its Git diff uses standard prefixes because this host's mnemonic diff prefixes otherwise hide changed functions. IO density stays visible at the explicit load/output boundaries; selector state and reporter accumulation still require design judgment.

The highest-ranked `next` candidate is a `chop` in `evals/skill/main.go:140` (`check`). Its work intentionally verifies the edited document and then the rules. Other candidates point at uniqueness state, recursive reference resolution, path labels, and selector dependencies. This rerun makes no further implementation changes; it records the evidence for a separate design review.

Build, all Go race tests, pinned lint, and all eight npm smoke tests under both Bun and Node pass again. CC/COG remains 7/6 for `RunLint` and 8/9 for `TextReporter.Format`.

Additional rerun commands:

```sh
cyclo check --tests --format json .
GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=diff.mnemonicPrefix GIT_CONFIG_VALUE_0=false cyclo check --changed --base f8d4549 --format json .
cyclo score .
cyclo next .
```

The earlier sections record the initial tool's results and review decisions.

## Changes

- Fix unbounded recursion in `selector.selectSegments`. A terminal descendant that cannot use schema-aware field selection now returns JSONPath value targets. `$..name` without an index and `$..*`, descendant unions, array indexes, and filters can otherwise repeat the same recursive call until the process stack overflows. New tests cover the fallback with and without an index, plus compound queries without an index. Running the new fallback test against the original source through a Go overlay produces `fatal error: stack overflow`; the fixed source passes.
- Split `RunLint` into document preparation, rule loading, rule evaluation, and reporting. Keep a single shared document index and preserve rule counts, severity handling, diagnostic text, and error behavior. Resolved rules stay in a local map instead of replacing `RulesWrapper.Rules`.
- Split text reporting into row collection, bucket ordering, column sizing, and output. Bucket ordering belongs to `bucket.less`; bucket output belongs to `bucket.write`. Existing group-order, alignment, truncation, color, and summary tests pass. A new test checks every write boundary, including the all-passed output, and confirms that writes stop at the first error.
- Remove the nil-map assignment from `RulesWrapper.ResolvedRules`. Resolution returns a new map and leaves the loaded wrapper unchanged. Extend the inherited-rules test to verify this behavior.

## Complexity and quality

Cyclo reports these CC/COG values. Baseline values use the same `gocyclo` and `gocognit` engines as Cyclo.

| Function | Before CC / COG | After CC / COG |
| --- | --- | --- |
| `RunLint` | 19 / 27 | 7 / 6 |
| `TextReporter.Format` | 34 / 62 | 8 / 9 |

The extracted lint helpers peak at CC 6 / COG 8. The extracted reporter helpers peak at CC 9 / COG 14. These changes separate tasks and state ownership; total branch count does not disappear.

The full production scan includes `evals/skill` and excludes `_test.go`. It has **72 findings before and 70 after**. The check still exits 1; this is not a clean quality gate. No suppression or threshold change is added.

| Rule | Before | After |
| --- | ---: | ---: |
| `fn_length` | 5 | 3 |
| `fn_params` | 2 | 2 |
| `mutated_targets` | 10 | 9 |
| `mutation_per_target` | 12 | 10 |
| `aggregate` | 2 | 1 |
| `side_effect_density` | 41 | 45 |

The additional density findings expose the extracted IO boundaries (`RunLint`, document loading, rule loading, reporter output). IO stays in those boundaries. Reporter collection still uses counters and mutable buckets; extraction does not establish purity. The after scan has 64 incomplete function summaries, primarily from third-party calls and dynamic dispatch. Unknown effects remain visible.

Remaining findings include uniqueness state, path-label construction, rule merging/execution, the recursive reference resolver, selector state, and the local skill-evaluation harness. They need separate design decisions. This PR addresses the crash and the two large orchestration functions; it does not claim to remove all quality debt.

## Pattern decisions

`cyclo patterns .` reports 21 candidates before and after. `cyclo fix --kind all .` reports zero fixable candidates. An apply run makes no edits. A pattern candidate is a review signal, not proof of a defect.

| Pattern | Count | Review decision |
| --- | ---: | --- |
| `typednil` | 13 | Target checks use concrete `*selector.Target`; schema checks use concrete maps; the registry uses a concrete function type. These nil comparisons are valid. The resolver receives JSON-decoded values, which do not carry typed nil pointers. Keep the guards. |
| `forcetypeassert` | 2 | `SchemaRule.cache` is private and `schemaFor` is its only writer. It stores compiled `*jsonschema.Schema` values. Keep this invariant rather than add unreachable error handling. |
| `ccgraph_clone` | 3 | `main` and registry initialization do different work. Version loaders use separate `sync.Once` state. `mapOf` and `stringOf` have different concrete return types. No useful shared helper is needed. |
| `value_object` | 1 | The evaluation harness passes workspace/bin paths together. A context type may help if this harness grows; it is outside the linter execution change. |
| `primitive_obsession` | 1 | A display group name and a filesystem filename have different meanings. Sharing a `Name` type would weaken their boundaries. |
| `parameterize` | 1 | Document decoding, rule merging, and filesystem path preparation do different work. A generic callback pipeline would hide the explicit error paths. |

## Verification and review notes

Commands:

```sh
cyclo --skill
cyclo patterns .
cyclo fix --kind all .
cyclo fix --kind all --apply .
cyclo check --format json .
go test -race ./...
go build ./...
GOFLAGS=-buildvcs=false make lint
OPENRPC_LINTER_TEST_BINARY=/path/to/built/binary bun test npm/test/shim.test.mjs npm/test/versions.test.mjs
OPENRPC_LINTER_TEST_BINARY=/path/to/built/binary node --test npm/test/shim.test.mjs npm/test/versions.test.mjs
```

The pinned linter needs network access for Go module resolution. Local lint uses `-buildvcs=false` because its Go 1.24 subprocess cannot obtain VCS status in this worktree; PR CI uses its normal `make lint` command. Node smoke tests need process access outside the restricted sandbox. No dependencies are added.

Cyclo's local annotation store contains the human request (marked as a request, not human approval) and agent review notes for the selector, lint flow, reporter, rule resolution, and schema-cache invariant. The TUI is left on the fixed selector. This session is outside Herdr, so no Herdr panes are controlled. `gherky` is not installed; the repository's build and test commands provide verification.
