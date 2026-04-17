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
