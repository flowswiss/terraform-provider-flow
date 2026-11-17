# Changelog

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


