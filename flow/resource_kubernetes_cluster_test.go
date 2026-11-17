package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccKubernetesCluster_Lifecycle(t *testing.T) {
	clusterName := acctest.RandomWithPrefix("test-cluster")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterConfig(clusterName, 3, "k1.1x2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "id"),
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "name", clusterName),
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "location_id", "1"),
					resource.TestCheckResourceAttrPair("flow_kubernetes_cluster.foobar", "network_id", "flow_compute_network.foobar", "id"),
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "security_group_id"),
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "public", "true"),
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "public_address"),
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "dns_name"),
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "version_id"),
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "node_count", "3"),
					resource.TestCheckResourceAttrPair("flow_kubernetes_cluster.foobar", "node_product_id", "data.flow_product.node", "id"),

					resource.TestCheckResourceAttrSet("data.flow_kubernetes_kube_config.foobar", "kube_config"),

					resource.TestCheckResourceAttrPair("data.flow_kubernetes_node.control_plane", "cluster_id", "flow_kubernetes_cluster.foobar", "id"),
					resource.TestCheckResourceAttrSet("data.flow_kubernetes_node.control_plane", "id"),
					resource.TestCheckResourceAttrSet("data.flow_kubernetes_node.control_plane", "name"),
					resource.TestCheckTypeSetElemAttr("data.flow_kubernetes_node.control_plane", "roles.*", "control-plane"),
					resource.TestCheckResourceAttrSet("data.flow_kubernetes_node.control_plane", "product_id"),
					resource.TestCheckResourceAttrPair("data.flow_kubernetes_node.control_plane", "network_id", "flow_compute_network.foobar", "id"),
					resource.TestCheckResourceAttrSet("data.flow_kubernetes_node.control_plane", "private_ip"),
					resource.TestCheckResourceAttrSet("data.flow_kubernetes_node.control_plane", "status"),
				),
			},
			{
				// upsize
				Config: testAccKubernetesClusterConfig(clusterName, 4, "k1.1x2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("flow_kubernetes_cluster.foobar", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "node_count", "4"),
			},
			{
				// downsize
				Config: testAccKubernetesClusterConfig(clusterName, 3, "k1.1x2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("flow_kubernetes_cluster.foobar", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "node_count", "3"),
			},
			{
				// flavor change
				Config: testAccKubernetesClusterConfig(clusterName, 3, "k1.2x2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("flow_kubernetes_cluster.foobar", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "node_count", "3"),
					resource.TestCheckResourceAttrPair("flow_kubernetes_cluster.foobar", "node_product_id", "data.flow_product.node", "id"),
				),
			},
			{
				ResourceName:      "flow_kubernetes_cluster.foobar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesClusterConfig(clusterName, 3, "k1.2x2") + testAccKubernetesClusterConfigDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_kubernetes_cluster.by_id", "id", "flow_kubernetes_cluster.foobar", "id"),
					resource.TestCheckResourceAttrPair("data.flow_kubernetes_cluster.by_id", "name", "flow_kubernetes_cluster.foobar", "name"),
					resource.TestCheckResourceAttrPair("data.flow_kubernetes_cluster.by_id", "location_id", "flow_kubernetes_cluster.foobar", "location_id"),
					resource.TestCheckResourceAttrPair("data.flow_kubernetes_cluster.by_id", "network_id", "flow_kubernetes_cluster.foobar", "network_id"),
					resource.TestCheckResourceAttrPair("data.flow_kubernetes_cluster.by_id", "security_group_id", "flow_kubernetes_cluster.foobar", "security_group_id"),
					resource.TestCheckResourceAttrPair("data.flow_kubernetes_cluster.by_id", "public_address", "flow_kubernetes_cluster.foobar", "public_address"),
					resource.TestCheckResourceAttrPair("data.flow_kubernetes_cluster.by_id", "dns_name", "flow_kubernetes_cluster.foobar", "dns_name"),
					resource.TestCheckResourceAttrPair("data.flow_kubernetes_cluster.by_id", "version_id", "flow_kubernetes_cluster.foobar", "version_id"),
					resource.TestCheckResourceAttrPair("data.flow_kubernetes_cluster.by_id", "node_count", "flow_kubernetes_cluster.foobar", "node_count"),
					resource.TestCheckResourceAttrPair("data.flow_kubernetes_cluster.by_id", "node_product_id", "flow_kubernetes_cluster.foobar", "node_product_id"),
					resource.TestCheckResourceAttrPair("data.flow_kubernetes_cluster.by_name", "id", "flow_kubernetes_cluster.foobar", "id"),
				),
			},
		},
	})
}

const testAccKubernetesClusterConfigDataSource = `
data "flow_kubernetes_cluster" "by_id" {
	id = flow_kubernetes_cluster.foobar.id
}

data "flow_kubernetes_cluster" "by_name" {
	name = flow_kubernetes_cluster.foobar.name
}
`

func testAccKubernetesClusterConfig(name string, nodeCount int, nodeProduct string) string {
	return fmt.Sprintf(`
data "flow_product" "node" {
	name = "%[3]s"
}

resource "flow_compute_network" "foobar" {
	name        = "%[1]s"
	location_id = 1
	cidr        = "10.108.0.0/24"
}

resource "flow_compute_router" "foobar" {
	name        = "%[1]s"
	location_id = 1
	public      = true
}

resource "flow_compute_router_interface" "foobar" {
	router_id  = flow_compute_router.foobar.id
	network_id = flow_compute_network.foobar.id
}

resource "flow_kubernetes_cluster" "foobar" {
	name = "%[1]s"

	location_id = 1
	network_id  = flow_compute_network.foobar.id

	public = true

	node_count      = %[2]d
	node_product_id = data.flow_product.node.id

	depends_on = [flow_compute_router_interface.foobar]
}

# the cluster's create returns once it is healthy, so both reads work right away
data "flow_kubernetes_kube_config" "foobar" {
	cluster_id = flow_kubernetes_cluster.foobar.id
}

# the control plane is the one role a cluster has exactly once
data "flow_kubernetes_node" "control_plane" {
	cluster_id = flow_kubernetes_cluster.foobar.id
	role       = "control-plane"
}
`, name, nodeCount, nodeProduct)
}
