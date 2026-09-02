# Changelog

## Unreleased (planned as v1.1.3)

### Behaviour changes
- `flow_compute_server.product_id` resizes the server in place (stop, resize, start — about a minute
  of downtime) instead of replacing it.
- `flow_compute_volume_attachment.volume_id` replaces the attachment (detach, attach).
- `flow_compute_network_interface.security_group_ids` is a set instead of a list; existing configs
  and states keep working.
- `flow_compute_volume.name` is required — the api refuses a create without one, so the error moves
  from a failed apply to the plan.

### Reliability
- Mutating API calls are retried with a bounded backoff instead of failing on the first transient
  error (new provider attribute `retry_timeout`, default `90s`, `0` disables); reads are retried on
  gateway errors.
- The provider waits until resources are actually usable — servers running, volumes settled, load
  balancers mutable, clusters ready or gone — with a deadline and a clear error on every wait.
- A create that fails halfway leaves a tainted resource instead of an untracked one, and objects
  deleted outside Terraform are dropped from the state and recreated instead of failing every plan.
- A 404 counts as success only for deletes and detaches; on updates the api's error surfaces —
  the api answers 404 for missing sub-entities too (e.g. an unknown cluster version), which was
  silently mistaken for success before.

### Fixes and additions
- Kubernetes clusters can be updated without pinning `version_id`, and a version change keeps the
  cluster's configuration variables.
- Renames no longer rebuild dependent resources such as routes, pool members and elastic-IP
  attachments.
- `key_pair_id` and `network_id` may be omitted; the values the api assigns are adopted cleanly.
- `flow_compute_server` exposes `network_interface_id` and `security_group_ids`: security groups on
  the server's own interface, and elastic-IP attachments without a data-source lookup.
- Every resource can be imported; resources that live under a parent use composite ids such as
  `server_id:id` (the format is in each resource's documentation). Key pairs and certificates carry
  attributes the api never returns (`public_key`, `certificate`, `private_key`), so the plan after
  their import wants a replace — and a replaced key pair rebuilds the servers referencing it; add
  `lifecycle { ignore_changes = [public_key] }` (or `certificate`, `private_key`) to adopt them.

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


