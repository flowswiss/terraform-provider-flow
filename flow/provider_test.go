package flow

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/flowswiss/goclient/v2/kubernetes"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var protoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"flow": providerserver.NewProtocol6WithError(New(
		WithVersion("test"),
	)),
}

// testAccSequential runs a test case without t.Parallel() — the api cannot
// create two billable objects for one org at the same time (metrics table,
// "Record has changed since last read"), a parallel run fails at random.
// Back to resource.ParallelTest once that is fixed.
func testAccSequential(t *testing.T, testCase resource.TestCase) {
	t.Helper()
	testCase.CheckDestroy = testAccCheckDestroyed
	resource.Test(t, testCase)
}

const destroyCheckTimeout = 2 * time.Minute

// children and attachments are left out: after destroy their parent is gone
// too, and the api no longer answers for them on its own
var testAccExists = map[string]func(ctx context.Context, client flowClient, id int) (bool, error){
	"flow_compute_certificate": func(ctx context.Context, client flowClient, id int) (bool, error) {
		list, err := client.Compute.Certificate.List(ctx, core.CursorAll)
		if err != nil {
			return false, err
		}
		for _, certificate := range list.Items {
			if certificate.ID == id {
				return true, nil
			}
		}
		return false, nil
	},
	"flow_compute_elastic_ip": func(ctx context.Context, client flowClient, id int) (bool, error) {
		_, found, err := findComputeElasticIP(ctx, client.Compute.ElasticIP, id)
		return found, err
	},
	"flow_compute_key_pair": func(ctx context.Context, client flowClient, id int) (bool, error) {
		list, err := client.Compute.KeyPair.List(ctx, core.CursorAll)
		if err != nil {
			return false, err
		}
		for _, keyPair := range list.Items {
			if keyPair.ID == id {
				return true, nil
			}
		}
		return false, nil
	},
	"flow_compute_load_balancer": func(ctx context.Context, client flowClient, id int) (bool, error) {
		_, err := client.Compute.LoadBalancer.Get(ctx, compute.LoadBalancerGetReq{ID: uint(id)})
		return existsFromGet(err)
	},
	"flow_compute_network": func(ctx context.Context, client flowClient, id int) (bool, error) {
		_, err := client.Compute.Network.Get(ctx, compute.NetworkGetReq{ID: uint(id)})
		return existsFromGet(err)
	},
	"flow_compute_router": func(ctx context.Context, client flowClient, id int) (bool, error) {
		_, err := client.Compute.Router.Get(ctx, compute.RouterGetReq{ID: uint(id)})
		return existsFromGet(err)
	},
	"flow_compute_security_group": func(ctx context.Context, client flowClient, id int) (bool, error) {
		_, err := client.Compute.SecurityGroup.Get(ctx, compute.SecurityGroupGetReq{ID: uint(id)})
		return existsFromGet(err)
	},
	"flow_compute_server": func(ctx context.Context, client flowClient, id int) (bool, error) {
		_, err := client.Compute.Server.Get(ctx, compute.ServerGetReq{ID: uint(id)})
		return existsFromGet(err)
	},
	"flow_compute_snapshot": func(ctx context.Context, client flowClient, id int) (bool, error) {
		_, err := client.Compute.Snapshot.Get(ctx, compute.SnapshotGetReq{ID: uint(id)})
		return existsFromGet(err)
	},
	"flow_compute_volume": func(ctx context.Context, client flowClient, id int) (bool, error) {
		_, err := client.Compute.Volume.Get(ctx, compute.VolumeGetReq{ID: uint(id)})
		return existsFromGet(err)
	},
	"flow_kubernetes_cluster": func(ctx context.Context, client flowClient, id int) (bool, error) {
		_, err := client.Kubernetes.Cluster.Get(ctx, kubernetes.ClusterGetReq{ID: uint(id)})
		return existsFromGet(err)
	},
}

func existsFromGet(err error) (bool, error) {
	if isNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

// testAccCheckDestroyed fails the test when the api still has an object after
// terraform destroyed it — a destroy that only drops the state stays green otherwise
func testAccCheckDestroyed(state *terraform.State) error {
	client, err := testAccClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	for name, rs := range state.RootModule().Resources {
		exists, ok := testAccExists[rs.Type]
		if !ok {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("%s: id %q is not numeric", name, rs.Primary.ID)
		}

		err = waitFor(ctx, destroyCheckTimeout, defaultWaitInterval, fmt.Sprintf("%s (%d) to be gone", name, id), func(ctx context.Context) (bool, error) {
			still, err := exists(ctx, client, id)
			return !still, err
		})
		if err != nil {
			return fmt.Errorf("%s still exists after destroy: %w", name, err)
		}
	}
	return nil
}

func testAccClient() (flowClient, error) {
	endpoint := "https://api.flow.swiss/"
	if val, ok := os.LookupEnv("FLOW_ENDPOINT"); ok {
		endpoint = val
	}

	baseURL, err := url.Parse(endpoint)
	if err != nil {
		return flowClient{}, fmt.Errorf("invalid FLOW_ENDPOINT: %w", err)
	}

	return newFlowClient(core.ClientOpts{
		BaseURL:    baseURL,
		HTTPClient: newHTTPClient(),
		UserAgent:  "terraform-provider-flow/test",
		Token:      os.Getenv("FLOW_TOKEN"),
	}), nil
}

func testAccServerConfig(t *testing.T, name string, cidr string) string {
	t.Helper()

	publicKey, _, err := acctest.RandSSHKeyPair(name)
	if err != nil {
		t.Fatal(err)
	}

	return fmt.Sprintf(`
data "flow_compute_image" "ubuntu" {
	key = "linux-ubuntu-22.04-lts"
}

data "flow_product" "small" {
	name = "b1.1x1"
}

resource "flow_compute_key_pair" "foobar" {
	name       = "%[1]s"
	public_key = "%[3]s"
}

resource "flow_compute_network" "foobar" {
	name        = "%[1]s"
	location_id = 1
	cidr        = "%[2]s"
}

resource "flow_compute_server" "foobar" {
	name        = "%[1]s"
	location_id = 1
	image_id    = data.flow_compute_image.ubuntu.id
	product_id  = data.flow_product.small.id
	network_id  = flow_compute_network.foobar.id
	key_pair_id = flow_compute_key_pair.foobar.id
}
`, name, cidr, publicKey)
}

func testAccCompositeImportID(name string, attributes ...string) resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		rs, ok := state.RootModule().Resources[name]
		if !ok {
			return "", fmt.Errorf("%s not in state", name)
		}

		parts := make([]string, len(attributes))
		for i, attribute := range attributes {
			parts[i] = rs.Primary.Attributes[attribute]
		}
		return strings.Join(parts, ":"), nil
	}
}
