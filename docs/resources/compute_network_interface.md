---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "flow_compute_network_interface Resource - terraform-provider-flow"
subcategory: ""
description: |-
  
---

# flow_compute_network_interface (Resource)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `network_id` (Number) unique identifier of the network
- `server_id` (Number) unique identifier of the server

### Optional

- `private_ip` (String) private IP address of the network interface
- `security` (Boolean) whether to enable security groups on the network interface
- `security_group_ids` (List of Number) list of security group IDs to assign to the network interface

### Read-Only

- `id` (Number) unique identifier of the network interface
- `mac_address` (String) MAC address of the network interface


