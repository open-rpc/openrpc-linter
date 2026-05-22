# OpenRPC Linter

[![CI](https://github.com/open-rpc/openrpc-linter/workflows/CI/badge.svg)](https://github.com/open-rpc/openrpc-linter/actions)

Fast, extensible linter for OpenRPC documents.

## Usage

```bash
# Lint with default rules
openrpc-linter lint openrpc.json -r rules.yml

# Create a basic rules.yml using the recommended rules
openrpc-linter init

# JSON output
openrpc-linter lint openrpc.json -r rules.yml -f json

# Validate document structure
openrpc-linter validate openrpc.json
```

### Lint output

Text violations are grouped by their most-specific document anchor: one section per RPC method, per top-level `components.schemas/contentDescriptors/tags` entry, or per top-level OpenRPC section (`info`, etc.). Anything that doesn't match falls into a `general` bucket. Each row shows `path / level / message / rule`; the source file and a summary are printed once at the top and the summary is repeated at the bottom.

Column max widths (longer values truncate with `…`): path 48, severity 7, message 56, rule 32. Secondary anchors (param name, schema title, descriptor name, tag) that aren't the group key render as indented continuation lines underneath the row.

```text
openrpc.json

7 errors, 8 warnings found in 8 rules

  path                                     level    message                                                   rule

debug_getBadBlocks
  methods[0].description                   error    missing field 'description'                               method-description
  methods[0].result.schema.description     warning  missing field 'description'                               schema-description
    schema: "Bad block"

debug_getRawBlock
  methods[1].params[0].schema.description  warning  missing field 'description'                               schema-description
    param: "n"
    schema: "Block"

schema "Pet"
  components.schemas.Pet.title             warning  missing field 'title'                                     schema-title

info
  info.license                             warning  missing field 'license'                                   info-license

7 errors, 8 warnings found in 8 rules
```

Group ordering is stable: method groups first (ordered by their `methods[N]` document position), then component `schema`/`descriptor`/`tag` groups (alphabetical), then other top-level sections, then `general`.

When stdout is a TTY the row is colored to give the severity column visual priority:

- `error` bright red, `warning` yellow, `info` blue
- path: dark grey
- message: light grey
- rule id: dark grey
- secondary continuation lines (`schema: "..."`, `param: "..."`): dark grey

Colors are skipped when piped or redirected, when `NO_COLOR` is set, or when `TERM=dumb`. Set `FORCE_COLOR=1` (or `CLICOLOR_FORCE=1`) to opt back in for piped output.

JSON output is unchanged (flat list with `pathLabels` alongside the canonical `path` array); clients can group themselves.

## Install

```bash
go install github.com/open-rpc/openrpc-linter@latest
```

## Rules

Create a rules `rules.yml` with rules you want to apply:

```yaml
extends:
  - recommended
```

Or define custom rules:

```yaml
rules:
  method-description:
    description: "Methods must have descriptions"
    given: "$.methods[*].description"
    severity: "error"
    then:
      function: "truthy"
```

The built-in functions currently include:

- `truthy`: require a selected value to be present and non-empty
- `unique`: require every selected value to be distinct across one rule run
- `referenced`: require selected components to have an inbound internal `$ref`

`unique` is symmetric with `truthy`: point `given` directly at the values you want to compare. By default duplicates are tracked in a single global bucket per rule run. Set `functionOptions.scope` to a JSONPath to partition duplicates by the longest-matching scope; targets outside every scope match are skipped (not deduped globally).

```yaml
rules:
  unique-method-names:
    description: "Method names must be unique"
    given: "$.methods[*].name"
    severity: "error"
    then:
      function: "unique"

  unique-param-names-per-method:
    description: "Param names should be unique within each method"
    given: "$.methods[*].params[*].name"
    severity: "error"
    then:
      function: "unique"
      functionOptions:
        scope: "$.methods[*]"
```

`unique` also supports `then.functionOptions.ignoreMissing`, which defaults to `true` and only matters for `given` paths whose terminal segment is a field name (so the selector can emit missing-field targets).

Use `referenced` with a `given` path that selects components directly. It scans the original document for internal refs and counts refs from both inside and outside `#/components`, so shared components used only by other components are allowed.

```yaml
rules:
  unused-components:
    description: "Components should be referenced."
    given: "$.components.*[*]"
    severity: "warn"
    then:
      function: "referenced"
```
