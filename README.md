# Workbench Terraform Provider

## Authentication

The provider supports multiple authentication methods:

### CLI Credentials (Recommended for Local Development)

Use credentials from the Workbench CLI. This is the recommended approach for local development as it
works with enterprise SSO.

**Prerequisites:**

1. Install the Workbench CLI (`wb`)
2. Authenticate: `wb auth login`

**Provider configuration:**

```terraform
provider "workbench" {
  host                = "https://workbench.verily.com"
  use_cli_credentials = true
}
```

The provider calls `wb auth print-access-token` to get your access token, which is cached in memory
and automatically refreshed when expired.

### Application Default Credentials

Uses Google Application Default Credentials (ADC). This is the default if no authentication method
is specified.

```terraform
provider "workbench" {
  host = "https://workbench.verily.com"
}
```

Note that this may fail if SAM doesn't have the account's GoogleSubjectId (true for most users
created after 2025).

### Workload Identity (GKE)

For service-to-service authentication in GKE clusters:

```terraform
provider "workbench" {
  host         = "https://workbench.verily.com"
  use_id_token = true
}
```

### Service Account Impersonation

Impersonate a service account using ADC:

```terraform
provider "workbench" {
  host                         = "https://workbench.verily.com"
  impersonate_service_account  = "my-sa@project.iam.gserviceaccount.com"
}
```

To use service account impersonation, you must have the `iam.serviceAccountTokenCreator` GCP role
for that service account. This is not set by default. You should be able to grant it to yourself in
dev but it's not granted to users in prod.

## Local development

Run `go build -o terraform-provider-workbench .` to build an executable in /bin. You will see
`terrafrom-provider-workbench` in your bin folder. Note that it won't be able to find the provider
without the `-workbench` suffix so you must specify the output name to append the suffix.

### Prepare Terraform for local provider install

Terraform installs providers and verifies their versions and checksums when you run
`terraform init`. Terraform will download your providers from either the provider registry or a
local registry. However, while building your provider you will want to test Terraform configuration
against a local development build of the provider. The development build will not have an associated
version number or an official set of checksums listed in a provider registry.

Terraform allows you to use local provider builds by setting a dev_overrides block in a
configuration file called .terraformrc. This block overrides all other configured installation
methods.

Terraform searches for the `~/.terraformrc` file and applies any configuration settings you set.

First, find the GOBIN path where Go installs your binaries. Your path may vary depending on how your
Go environment variables are configured.

```bash
go env GOBIN
```

If the GOBIN go environment variable is not set, use the default path, `/Users/<Username>/go/bin`.

Create a new file called .terraformrc in your home directory (~), then add the dev_overrides block
below. Change the <PATH> to the value returned from the go env GOBIN command above.

```sh
provider_installation {

  dev_overrides {
      "registry.terraform.io/verily-src/workbench" = "<repo-path>/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

### Testing

To run acceptance test, set TF_ACC=1

```sh
TF_ACC=1 go test ./... -v
```

## Generate documentation

```bash
cd tools && go generate ./...
```
