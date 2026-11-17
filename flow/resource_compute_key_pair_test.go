package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeKeyPair_Basic(t *testing.T) {
	keyPairName := acctest.RandomWithPrefix("test-key-pair")
	public, _, err := acctest.RandSSHKeyPair("test-key-pair")
	if err != nil {
		t.Fatal(err)
	}

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeKeyPairConfigBasic, keyPairName, public),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_key_pair.foobar", "id"),
					resource.TestCheckResourceAttrSet("flow_compute_key_pair.foobar", "fingerprint"),
					resource.TestCheckResourceAttr("flow_compute_key_pair.foobar", "name", keyPairName),
					resource.TestCheckResourceAttr("flow_compute_key_pair.foobar", "public_key", public),
				),
			},
			{
				ResourceName:      "flow_compute_key_pair.foobar",
				ImportState:       true,
				ImportStateVerify: true,
				// the api returns the fingerprint only, not the key
				ImportStateVerifyIgnore: []string{"public_key"},
			},
			{
				Config: fmt.Sprintf(testAccComputeKeyPairConfigBasic, keyPairName, public) + testAccComputeKeyPairConfigDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_key_pair.by_id", "id", "flow_compute_key_pair.foobar", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_key_pair.by_id", "name", "flow_compute_key_pair.foobar", "name"),
					resource.TestCheckResourceAttrPair("data.flow_compute_key_pair.by_id", "fingerprint", "flow_compute_key_pair.foobar", "fingerprint"),
					resource.TestCheckResourceAttrPair("data.flow_compute_key_pair.by_name", "id", "flow_compute_key_pair.foobar", "id"),
				),
			},
		},
	})
}

const testAccComputeKeyPairConfigBasic = `
resource "flow_compute_key_pair" "foobar" {
	name        = "%s"
	public_key  = "%s"
}
`

const testAccComputeKeyPairConfigDataSource = `
data "flow_compute_key_pair" "by_id" {
	id = flow_compute_key_pair.foobar.id
}

data "flow_compute_key_pair" "by_name" {
	name = flow_compute_key_pair.foobar.name
}
`
