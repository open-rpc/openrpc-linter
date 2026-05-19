# OpenRPC Linter

[![CI](https://github.com/open-rpc/openrpc-linter/workflows/CI/badge.svg)](https://github.com/open-rpc/openrpc-linter/actions)

Fast, extensible linter for OpenRPC documents.

## Usage

```bash
# Lint with default rules
openrpc-linter lint openrpc.json -r rules.yml

# JSON output
openrpc-linter lint openrpc.json -r rules.yml -f json

# Validate document structure
openrpc-linter validate openrpc.json
```

## Install

```bash
go install github.com/open-rpc/openrpc-linter@latest
```

## Rules

Create a rules `rules.yml` with rules you want to apply:

```yaml
rules:
  method-description:
    description: "Methods must have descriptions"
    given: "$.methods[*]"
    severity: "error"
    then:
      field: "description"
      function: "truthy"
```

The built-in functions currently include:

- `truthy`: require a field or selected value to be present and non-empty
- `unique`: require a field value to be unique across the selected collection

Example `unique` rules:

```yaml
rules:
  unique-method-names:
    description: "Method names must be unique"
    given: "$.methods"
    severity: "error"
    then:
      field: "name"
      function: "unique"

  unique-param-names-per-method:
    description: "Param names should be unique within each method"
    given: "$.methods[*].params"
    severity: "error"
    then:
      field: "name"
      function: "unique"
```

`unique` supports `then.functionOptions.ignoreMissing`, which defaults to `true`.
