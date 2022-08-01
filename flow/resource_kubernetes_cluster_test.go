package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesCluster_Basic(t *testing.T) {
	networkName := "default"
	clusterName := acctest.RandomWithPrefix("test-cluster")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccKubernetesClusterConfigBasic, networkName, clusterName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "id"),
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "name", clusterName),
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "location_id", "1"),
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "network_id"),
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "security_group_id"),
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "public", "true"),
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "public_address"),
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "dns_name"),
					resource.TestCheckResourceAttrSet("flow_kubernetes_cluster.foobar", "version_id"),
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "node_count", "3"),
					resource.TestCheckResourceAttr("flow_kubernetes_cluster.foobar", "node_product_id", "44"),
				),
			},
		},
	})
}

const testAccKubernetesClusterConfigBasic = `
data "flow_compute_network" "foobar" {
	name = "%s"
}

resource "flow_kubernetes_cluster" "foobar" {
	name = "%s"

	location_id = data.flow_compute_network.foobar.location_id
	network_id 	= data.flow_compute_network.foobar.id

	public = true

	node_count = 3
	node_product_id = 44
}
`
