---
page_title: "Known limitations"
subcategory: ""
description: |-
  what the provider cannot do today, and what to do instead
---

# Known limitations

Most of these come from the api, not from the provider.

## Imports and secrets

| What | What to do |
|---|---|
| importing a `flow_compute_key_pair` plans a replace of it and of every `flow_compute_server` that uses it | `lifecycle { ignore_changes = [public_key] }` on the `flow_compute_key_pair` |
| a change to a write-only argument plans nothing: `certificate` and `private_key` on `flow_compute_certificate`, `password` and `cloud_init` on `flow_compute_server` | `terraform apply -replace=<address>` |
| `kube_config` of the `flow_kubernetes_kube_config` data source is stored in the state | keep the state in an encrypted backend |

## Kubernetes

| What | What to do |
|---|---|
| `version_id` on `flow_kubernetes_cluster` cannot be set at create | create the `flow_kubernetes_cluster` first, set `version_id` in a later apply |
| the configuration variables of a `flow_kubernetes_cluster` are not managed. a change of `version_id` sends the current variables along, and the apply fails when they do not fit the target version's schema | adjust the variables in the portal first, or change the version there, then apply again |
| the nodes, volumes and load balancers of a `flow_kubernetes_cluster` are read-only: `flow_kubernetes_node`, `flow_kubernetes_volume` and `flow_kubernetes_load_balancer` are data sources | manage them inside the cluster or in the portal |

## Compute

| What | What to do |
|---|---|
| shrinking a `flow_compute_volume` replaces it. the data is lost | back up the data first, or set `lifecycle { prevent_destroy = true }` |
| removing `certificate_id` from a `flow_compute_load_balancer_pool` fails the apply | `terraform apply -replace=<address>` on the `flow_compute_load_balancer_pool` |

## Provider

| What | What to do |
|---|---|
| a request the api rejects only fails after about 90 seconds | `retry_timeout = "0"` or `FLOW_RETRY_TIMEOUT=0` |
| Terraform below 1.11 cannot set the write-only arguments | upgrade Terraform |
