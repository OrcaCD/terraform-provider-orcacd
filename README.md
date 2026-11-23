# <div align="center"><img alt="OrcaCD logo" src="https://raw.githubusercontent.com/OrcaCD/orca-cd/f6fb48965df90a57a2b181e9e2b359653ae5c6b9/frontend/public/assets/logo-dark-256.png" width="100"/> <br>OrcaCD Terraform Provider</div>

<div align="center">
Manage <a href="https://orcacd.dev">OrcaCD</a> with Terraform and OpenTofu<br><br>

<a href="https://github.com/OrcaCD/terraform-provider-orcacd/releases"><picture><source media="(prefers-color-scheme: dark)" srcset="https://shieldcn.dev/github/OrcaCD/terraform-provider-orcacd/release.svg?logo=github&mode=dark&size=sm&variant=outline"><img alt="GitHub Release" src="https://shieldcn.dev/github/OrcaCD/terraform-provider-orcacd/release.svg?logo=github&mode=light&size=sm&variant=outline"></picture></a>
<a href="https://github.com/OrcaCD/terraform-provider-orcacd/blob/main/go.mod"><picture><source media="(prefers-color-scheme: dark)" srcset="https://shieldcn.dev/badge/dynamic/json.svg?url=https%3A%2F%2Fimg.shields.io%2Fgithub%2Fgo-mod%2Fgo-version%2FOrcaCD%2Fterraform-provider-orcacd.json&query=%24.message&label=go&logo=go&logoColor=%2300add8&mode=dark&size=sm&variant=outline"><img alt="Go Version" src="https://shieldcn.dev/badge/dynamic/json.svg?url=https%3A%2F%2Fimg.shields.io%2Fgithub%2Fgo-mod%2Fgo-version%2FOrcaCD%2Fterraform-provider-orcacd.json&query=%24.message&label=go&logo=go&logoColor=%2300add8&mode=light&size=sm&variant=outline"></picture></a>
<a href="https://github.com/OrcaCD/terraform-provider-orcacd/actions/workflows/ci.yml"><picture><source media="(prefers-color-scheme: dark)" srcset="https://shieldcn.dev/github/actions/workflow/status/OrcaCD/terraform-provider-orcacd/ci.yml.svg?branch=main&logo=githubactions&mode=dark&size=sm&variant=outline"><img alt="CI" src="https://shieldcn.dev/github/actions/workflow/status/OrcaCD/terraform-provider-orcacd/ci.yml.svg?branch=main&logo=githubactions&mode=light&size=sm&variant=outline"></picture></a>
<a href="https://github.com/OrcaCD/terraform-provider-orcacd/blob/main/LICENSE"><picture><source media="(prefers-color-scheme: dark)" srcset="https://shieldcn.dev/github/OrcaCD/terraform-provider-orcacd/license.svg?logo=opensourceinitiative&logoColor=%233da639&mode=dark&size=sm&variant=outline"><img alt="License" src="https://shieldcn.dev/github/OrcaCD/terraform-provider-orcacd/license.svg?logo=opensourceinitiative&logoColor=%233da639&mode=light&size=sm&variant=outline"></picture></a>
<a href="https://github.com/OrcaCD/terraform-provider-orcacd/issues"><picture><source media="(prefers-color-scheme: dark)" srcset="https://shieldcn.dev/github/OrcaCD/terraform-provider-orcacd/issues.svg?logo=github&mode=dark&size=sm&variant=outline"><img alt="GitHub Issues" src="https://shieldcn.dev/github/OrcaCD/terraform-provider-orcacd/issues.svg?logo=github&mode=light&size=sm&variant=outline"></picture></a>
</div>

Manage Git repositories, Compose applications, notification subscriptions, and agents on an OrcaCD hub with Terraform or OpenTofu. Every resource supports import by its hub ID and has a data source for lookup by ID or exact name.

- [OrcaCD setup guide](https://orcacd.dev/docs/guides/terraform)
- [Provider configuration](docs/index.md)
- [Quickstart](examples/quickstart/main.tf)
- [Resources](docs/resources) and [data sources](docs/data-sources)
- [Changelog](CHANGELOG.md)

The reference under `docs/` ships with each provider version for the Terraform and OpenTofu registries. The setup guide lives in the [OrcaCD docs repository](https://github.com/OrcaCD/docs).

## Try a local build

Use Go 1.27 or newer and an OrcaCD hub that supports bearer API tokens. Create a token with access to the resources you want to manage and set `ORCACD_URL` to the hub's base URL and `ORCACD_TOKEN` to the token.

```sh
go build -o bin/terraform-provider-orcacd .
```

Create a temporary CLI configuration file with the absolute path to the `bin` directory.

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/orcacd/orcacd" = "/absolute/path/to/bin"
    "registry.opentofu.org/orcacd/orcacd"  = "/absolute/path/to/bin"
  }
  direct {}
}
```

Set `TF_CLI_CONFIG_FILE` to that file for either Terraform or OpenTofu, then run the quickstart. With the development override in place, run `plan` and `apply` directly. Once a release is available from the registry, use normal provider installation via `init`.

## Contributing

```sh
go test -race ./...
go vet ./...
go build ./...
terraform fmt -check -recursive
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.25.0 validate --provider-name orcacd
```

The checked-in HTTP client is generated from the OrcaCD OpenAPI export in [OrcaCD/orca-cd#576](https://github.com/OrcaCD/orca-cd/pull/576). It targets the hub's `/api/v1` endpoints. Use `go generate ./internal/api` with `curl` and `jq` to regenerate it. The script pins the OrcaCD commit and restores the embedded agent fields present in the server's creation response.

## Releasing

Release Please uses Conventional Commits such as `feat:` and `fix:` on `main` to propose a version and update [CHANGELOG.md](CHANGELOG.md) in a release pull request. The prepare workflow uses the OrcaCD GitHub App so the release PR can run CI. Install the App for this repository with write access to contents and pull requests. In the repository's `release` environment, set `APP_ID` to the App's Client ID and set `APP_PRIVATE_KEY` to its PEM private key.

Review the changelog in the release PR. Merging the PR uses the same GitHub App to push the proposed version tag, such as `v0.1.0`, at the merge commit. The release workflow clears the PR's pending label after publishing the signed release so Release Please can propose the next version.

Before merging the first release PR, choose an RSA GPG signing key for registry compatibility. Set the `GPG_PRIVATE_KEY` repository Actions secret to the ASCII-armored private key and `GPG_PASSPHRASE` to its passphrase. The release workflow builds provider ZIP files, the protocol manifest, and signed SHA-256 checksums on a version tag. After the first signed release appears on GitHub, register the provider and its public signing key in each registry. See the [Terraform Registry publishing guide](https://developer.hashicorp.com/terraform/registry/providers/publishing) and [OpenTofu Registry submission instructions](https://github.com/opentofu/registry#adding-providers-modules-or-gpg-keys-to-the-opentofu-registry).
