---
page_title: "Update or replace"
subcategory: ""
description: |-
  which changes update a resource in place and which replace it
---

# Update or replace

A change either updates a resource in place or replaces it. Replace means destroy, then create.
The api decides which.

Computed attributes are not listed. They cannot be changed.

## Compute

| Resource | Updates in place | Replaces |
|---|---|---|
| `flow_compute_certificate` | — | `name`, `location_id` |
| `flow_compute_elastic_ip` | — | `location_id` |
| `flow_compute_elastic_ip_load_balancer_attachment` | — | `elastic_ip_id`, `load_balancer_id` |
| `flow_compute_elastic_ip_server_attachment` | — | `elastic_ip_id`, `network_interface_id`, `server_id` |
| `flow_compute_key_pair` | — | `name`, `public_key` |
| `flow_compute_load_balancer` | `name` | `location_id`, `network_id`, `private_ip` |
| `flow_compute_load_balancer_member` | — | `address`, `load_balancer_id`, `name`, `pool_id`, `port` |
| `flow_compute_load_balancer_pool` | `balancing_algorithm_id`, `certificate_id`, `health_check`, `sticky_session` | `entry_port`, `entry_protocol_id`, `load_balancer_id`, `target_protocol_id` |
| `flow_compute_network` | `name`, `allocation_pool`, `domain_name_servers`, `gateway_ip` | `cidr`, `location_id` |
| `flow_compute_network_interface` | `security`, `security_group_ids` | `network_id`, `private_ip`, `server_id` |
| `flow_compute_router` | `name`, `public` | `location_id` |
| `flow_compute_router_interface` | — | `network_id`, `private_ip`, `router_id` |
| `flow_compute_router_route` | — | `destination`, `next_hop`, `router_id` |
| `flow_compute_security_group` | `name` | `location_id` |
| `flow_compute_security_group_rule` | `direction`, `icmp`, `ip_range`, `port_range`, `protocol`, `remote_security_group_id` | `security_group_id` |
| `flow_compute_server` | `name`, `product_id`, `security_group_ids` | `image_id`, `key_pair_id`, `location_id`, `network_id`, `private_ip` |
| `flow_compute_snapshot` | `name` | `volume_id` |
| `flow_compute_volume` | `name`, `size` when it grows | `location_id`, `restore_from_snapshot_id`, `size` when it shrinks |
| `flow_compute_volume_attachment` | `server_id` | `volume_id` |

## Kubernetes

| Resource | Updates in place | Replaces |
|---|---|---|
| `flow_kubernetes_cluster` | `name`, `node_count`, `node_product_id`, `version_id` | `location_id`, `network_id`, `public` |

## Worth knowing

- `product_id` on `flow_compute_server` updates in place. The server is down for about a minute.
- `server_id` on `flow_compute_volume_attachment` moves the volume to the other server.
- A shrinking `flow_compute_volume` is replaced. The data is lost.
- A replaced `flow_compute_key_pair` replaces every `flow_compute_server` that uses it.
- `version_id` on `flow_kubernetes_cluster` only follows the upgrade paths of the current version.
- A change to `timeouts {}` alone does not touch the resource.
- Write-only arguments are not in the tables. A change to them plans nothing, see
  [known limitations](https://registry.terraform.io/providers/flowswiss/flow/latest/docs/guides/known-limitations).
