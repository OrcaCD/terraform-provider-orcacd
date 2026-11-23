# OrcaCD Terraform provider

Manage Git repositories, Compose applications, notification subscriptions, and agents on an OrcaCD hub with Terraform or OpenTofu. Every resource supports import by its hub ID and has a data source for lookup by ID or exact name.

- [Provider configuration](docs/index.md)
- [Quickstart](examples/quickstart/main.tf)
- [Resources](docs/resources) and [data sources](docs/data-sources)

## Try a local build

Use Go 1.26 or newer and an OrcaCD hub that supports bearer API tokens. Create a token with access to the resources you want to manage and set `ORCACD_URL` to the hub's base URL and `ORCACD_TOKEN` to the token.

```sh
go build -o bin/terraform-provider-orcacd .
```

Create a temporary Terraform CLI configuration file with the absolute path to the `bin` directory.

```hcl
provider_installation {
  dev_overrides {
    "orcacd/orcacd"                       = "/absolute/path/to/bin"
    "registry.opentofu.org/orcacd/orcacd" = "/absolute/path/to/bin"
  }
  direct {}
}
```

Run the quickstart with `TF_CLI_CONFIG_FILE` pointing at that file. With the development override in place, run `plan` and `apply` directly. Once a release is available from the registry, use normal provider installation via `init`.

## Contributing

```sh
go test -race ./...
go vet ./...
go build ./...
```

The checked-in HTTP client is generated from the OrcaCD OpenAPI export in [OrcaCD/orca-cd#576](https://github.com/OrcaCD/orca-cd/pull/576). It targets the hub's `/api/v1` endpoints. Use `go generate ./internal/api` with `curl` and `jq` to regenerate it. The script pins the OrcaCD commit and restores the embedded agent fields present in the server's creation response.

The release workflow builds provider ZIP files, the protocol manifest, and signed SHA-256 checksums on a version tag. Set the `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE` repository secrets for the release signing key. The first release requires provider registration and the public signing key in each registry. See the [Terraform Registry publishing guide](https://developer.hashicorp.com/terraform/registry/providers/publishing) and [OpenTofu Registry submission instructions](https://github.com/opentofu/registry#adding-providers-modules-or-gpg-keys-to-the-opentofu-registry).
