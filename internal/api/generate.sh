#!/bin/sh
set -eu

# OrcaCD's OpenAPI export from OrcaCD/orca-cd#576.
revision=5fcd17efa702c9a0e3ac42919c6731e4cabec23c
spec_dir=$(mktemp -d)
trap 'rm -rf "$spec_dir"' EXIT
curl -fsSL "https://raw.githubusercontent.com/OrcaCD/orca-cd/$revision/backend/openapi.json" -o "$spec_dir/upstream.json"

# Restore the embedded agent fields present in the create response. Keep the
# generated models focused on declared fields, ignoring extra fields in responses.
jq '
  .components.schemas.AgentWithTokenResponse as $token |
  .components.schemas.AgentResponse as $agent |
  .components.schemas.AgentWithTokenResponse = (
    $token + {
      properties: ($agent.properties + $token.properties),
      required: $agent.required
    }
  ) |
  .components.schemas |= with_entries(
    if .value.additionalProperties == true and .value.properties then
      .value |= del(.additionalProperties)
    else
      .
    end
  )
' "$spec_dir/upstream.json" > "$spec_dir/openapi.json"

go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.9.0 -config codegen.yaml "$spec_dir/openapi.json"
