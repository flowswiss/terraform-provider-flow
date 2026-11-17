package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*computeElasticIPLoadBalancerAttachmentResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeElasticIPLoadBalancerAttachmentResource)(nil)
	_ resource.ResourceWithImportState = (*computeElasticIPLoadBalancerAttachmentResource)(nil)
)

type computeElasticIPLoadBalancerAttachmentResourceData struct {
	LoadBalancerID types.Int64 `tfsdk:"load_balancer_id"`
	ElasticIPID    types.Int64 `tfsdk:"elastic_ip_id"`
}

func (c *computeElasticIPLoadBalancerAttachmentResourceData) FromEntity(loadBalancer compute.LoadBalancer, elasticIP compute.ElasticIP) {
	c.LoadBalancerID = types.Int64Value(int64(loadBalancer.ID))
	c.ElasticIPID = types.Int64Null()

	for _, network := range loadBalancer.Networks {
		for _, iface := range network.Interfaces {
			if iface.PublicIP != "" && iface.PublicIP == elasticIP.PublicIP {
				c.ElasticIPID = types.Int64Value(int64(elasticIP.ID))
			}
		}
	}
}

func (c computeElasticIPLoadBalancerAttachmentResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Attaches an elastic ip to a load balancer. A load balancer takes at most one, and the network it sits in has to be behind a public router.\n\nImport: `terraform import flow_compute_elastic_ip_load_balancer_attachment.<name> <load_balancer_id>:<elastic_ip_id>`",
		Attributes: map[string]schema.Attribute{
			"load_balancer_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer to attach the elastic ip to",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"elastic_ip_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the elastic ip to attach to the load balancer. It has to be free and at the load balancer's location",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func newComputeElasticIPLoadBalancerAttachmentResource() resource.Resource {
	return &computeElasticIPLoadBalancerAttachmentResource{}
}

func (c *computeElasticIPLoadBalancerAttachmentResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_elastic_ip_load_balancer_attachment"
}

func (c *computeElasticIPLoadBalancerAttachmentResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeElasticIPLoadBalancerAttachmentResource struct {
	client flowClient
}

func (c computeElasticIPLoadBalancerAttachmentResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeElasticIPLoadBalancerAttachmentResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.LoadBalancerElasticIPCreateReq{
		LoadBalancerID: uint(config.LoadBalancerID.ValueInt64()),
		ElasticIPID:    int(config.ElasticIPID.ValueInt64()),
	}

	err := retry(ctx, "attach elastic ip", func() error {
		_, err := c.client.Compute.LoadBalancerElasticIP.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to attach elastic ip: %s", err))
		return
	}

	diagnostics = response.State.Set(ctx, config)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeElasticIPLoadBalancerAttachmentResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeElasticIPLoadBalancerAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := state.LoadBalancerID.ValueInt64()
	elasticIPID := state.ElasticIPID.ValueInt64()

	loadBalancer, err := c.client.Compute.LoadBalancer.Get(ctx, compute.LoadBalancerGetReq{ID: uint(loadBalancerID)})
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("load balancer %d", loadBalancerID))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get load balancer: %s", err))
		return
	}

	elasticIP, found, err := findComputeElasticIP(ctx, c.client.Compute.ElasticIP, int(elasticIPID))
	if err != nil {
		response.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if !found {
		removeGone(ctx, response, fmt.Sprintf("elastic ip %d", elasticIPID))
		return
	}

	state.FromEntity(loadBalancer, elasticIP)

	if state.ElasticIPID.IsNull() {
		removeGone(ctx, response, fmt.Sprintf("attachment of elastic ip %d to load balancer %d", elasticIPID, loadBalancerID))
		return
	}

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeElasticIPLoadBalancerAttachmentResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	response.Diagnostics.AddError("Not Supported", "updating an elastic ip attachment is not supported")
}

func (c computeElasticIPLoadBalancerAttachmentResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeElasticIPLoadBalancerAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "detach elastic ip", func() error {
		return c.client.Compute.LoadBalancerElasticIP.Delete(ctx, compute.LoadBalancerElasticIPDeleteReq{LoadBalancerID: uint(state.LoadBalancerID.ValueInt64())})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to detach elastic ip: %s", err))
		return
	}
}

func (c computeElasticIPLoadBalancerAttachmentResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStateCompositeInt64IDs(ctx, request, response, path.Root("load_balancer_id"), path.Root("elastic_ip_id"))
}
