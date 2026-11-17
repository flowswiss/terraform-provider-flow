# Changelog

## Unreleased (planned as v1.2.0)

### Breaking
- Needs Terraform 1.11 or later.
- Mac bare metal is gone: `terraform state rm` takes them out of an existing state.
- These arguments are write-only now: `certificate` and `private_key` on a certificate, `password` and
  `cloud_init` on a server. Older Terraform cannot set them.

### New
- `flow_compute_snapshot` resource.
- `timeouts {}` on every resource that waits.
- `status` on `flow_compute_load_balancer_member` and its data source.
- `public_ip` on `flow_compute_load_balancer`.
- `flow_compute_elastic_ip_load_balancer_attachment` attaches your own elastic ip to a load balancer.
- `flow_kubernetes_node`, `flow_kubernetes_volume` and `flow_kubernetes_load_balancer` data sources introduced

### Fixes
- A resource the api created stays in the state when a later step fails.
- `false` and `0` reach the api.
- A router can be made private.
- The state matches the api after an update.
- Clearer errors for kubernetes version changes and an empty token. Several data source fixes.

### Dependencies
- framework v0.10.0 → v1.19.0, plugin-go v0.13.0 → v0.31.0, tests on plugin-testing v1.16.0.
  No behaviour change.
- New: terraform-plugin-framework-timeouts v0.7.0.
- CI runs on Terraform 1.11 and on the latest release.
- goclient v0.2.6 → v2.0.3.
- Acceptance tests check that nothing is left after a destroy, and cover import, update and the
  matching data source on every resource that has a test.

## Unreleased (planned as v1.1.3)

### Behaviour changes
- `flow_compute_server.product_id` resizes the server in place. About a minute of downtime.
- `flow_compute_volume_attachment.volume_id` replaces the attachment.
- `flow_compute_network_interface.security_group_ids` is a set, not a list. Existing configs keep
  working.
- `flow_compute_volume.name` is required.

### Reliability
- Failed api calls are retried. New provider attribute `retry_timeout`: default `90s`, `0` turns it
  off.
- The provider waits until a resource is really usable.
- A create that fails halfway leaves a tainted resource.
- A resource deleted outside Terraform is dropped from the state and created again.
- A 404 only counts as success on a delete or a detach.

### Fixes and additions
- A kubernetes cluster can be updated without setting `version_id`.
- A version change keeps the cluster's configuration variables.
- A rename no longer rebuilds dependent resources.
- `key_pair_id` and `network_id` can be left out.
- `network_interface_id` and `security_group_ids` on `flow_compute_server`.
- Every resource except `flow_mac_bare_metal_security_group_rule` can be imported. A resource under
  a parent uses an id like `server_id:id`.
- Importing a key pair or a certificate plans a replace. Add
  `lifecycle { ignore_changes = [public_key] }` (or `certificate`, `private_key`) to avoid it.

## v1.1.2 - 2026-08-17
- Fixed `terraform import` for every importable resource: all ids are numeric, but the import wrote
  them as strings and failed schema validation.
- Fixed the docs generation CI (tfplugindocs 0.13 → 0.25 — an expired GPG key broke every docs job),
  updated terraform-plugin-log to 0.11 and the GitHub Actions; the acceptance job is skipped when no
  token is configured so pull requests stop failing.

## v1.1.1 - 2025-11-11
- Require Go 1.25 for local builds.
- Verified dependency stack remains on legacy-compatible Terraform plugin libraries (framework v0.10.0, SDK v2.20.0).
- Added automated `govulncheck ./...` security scan to release pipeline.


