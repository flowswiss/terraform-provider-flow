# Flow Terraform Provider

Terraform provider for the [Flow Swiss](https://flow.swiss/) cloud: compute and kubernetes.

The docs are on the [Terraform Registry](https://registry.terraform.io/providers/flowswiss/flow/latest/docs).
Read the two guides first: update or replace, and known limitations.

## Requirements

- Terraform 1.11 or later
- Go 1.26 or later to build

## Developing

In order to develop the provider, you need to tell Terraform to use the locally built provider instead of fetching it
from the registry. To do this, add the following to your `~/.terraformrc` file:

```hcl
provider_installation {
  # Use the given directory as the provider installation directory.
  # This disables the version and checksum verifications for this
  # provider and forces Terraform to look for the provider plugin
  # in the given directory.
  dev_overrides {
    "flowswiss/flow" = "path to local `terraform-provider-flow` directory"
  }

  # For all other providers, use the default behavior.
  direct {}
}
```

Please see the
[Terraform Documentation](https://www.terraform.io/cli/config/config-file#development-overrides-for-provider-developers)
for more details about the `dev_overrides` section.

Once you have configured your `~/.terraformrc`, you must build the provider every time you change your code using
`go build .`. This generates the `terraform-provider-flow` binary which terraform can then use as a provider.

## Docs

`docs/` is generated. Never edit it by hand.

- Descriptions live in the schema.
- The provider page and the guides live in `templates/`.
- After a change run `go generate ./...` and commit the result. CI fails when `docs/` is out of date.
- Terraform 1.11 or later must be on the `PATH`. Without it the write-only markers drop out of the docs.

## Tests

`go test ./...` runs the unit tests.

The acceptance tests create real, billed resources. They only run with `TF_ACC` set:

```
TF_ACC=1 FLOW_TOKEN=… FLOW_ENDPOINT=… go test -v -count=1 -timeout 30m ./flow/
```

- Without `FLOW_ENDPOINT` they run against production.
- They run one at a time. The api cannot create two billed resources for one organisation at once.
