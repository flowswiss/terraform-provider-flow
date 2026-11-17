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
	_ resource.Resource                = (*computeElasticIPServerAttachmentResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeElasticIPServerAttachmentResource)(nil)
	_ resource.ResourceWithImportState = (*computeElasticIPServerAttachmentResource)(nil)
)

type computeElasticIPServerAttachmentResourceData struct {
	ServerID           types.Int64 `tfsdk:"server_id"`
	NetworkInterfaceID types.Int64 `tfsdk:"network_interface_id"`
	ElasticIPID        types.Int64 `tfsdk:"elastic_ip_id"`
}

func (c *computeElasticIPServerAttachmentResourceData) FromEntity(server compute.Server, elasticIP compute.ElasticIP) {
	c.ServerID = types.Int64Value(int64(server.ID))
	c.NetworkInterfaceID = types.Int64Null()
	c.ElasticIPID = types.Int64Value(int64(elasticIP.ID))

	for _, network := range server.Networks {
		for _, iface := range network.Interfaces {
			if iface.PublicIP == elasticIP.PublicIP {
				c.NetworkInterfaceID = types.Int64Value(int64(iface.ID))
			}
		}
	}
}

func (c computeElasticIPServerAttachmentResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Import: `terraform import flow_compute_elastic_ip_server_attachment.<name> <server_id>:<elastic_ip_id>`",
		Attributes: map[string]schema.Attribute{
			"server_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the server to attach the elastic ip to",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"network_interface_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network interface to attach the elastic ip to — `flow_compute_server.<name>.network_interface_id` for a server's primary interface",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"elastic_ip_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the elastic ip to attach to the server",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func newComputeElasticIPServerAttachmentResource() resource.Resource {
	return &computeElasticIPServerAttachmentResource{}
}

func (c *computeElasticIPServerAttachmentResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_elastic_ip_server_attachment"
}

func (c *computeElasticIPServerAttachmentResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeElasticIPServerAttachmentResource struct {
	client flowClient
}

func (c computeElasticIPServerAttachmentResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeElasticIPServerAttachmentResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	serverID := int(config.ServerID.ValueInt64())

	attach := compute.ElasticIPAttachmentCreateReq{
		ServerID:           uint(serverID),
		ElasticIPID:        int(config.ElasticIPID.ValueInt64()),
		NetworkInterfaceID: int(config.NetworkInterfaceID.ValueInt64()),
	}

	var elasticIP compute.ElasticIP
	err := retry(ctx, "attach elastic ip", func() (err error) {
		elasticIP, err = c.client.Compute.ElasticIPAttachment.Create(ctx, attach)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to attach elastic ip: %s", err))
		return
	}

	server, err := c.client.Compute.Server.Get(ctx, compute.ServerGetReq{ID: uint(serverID)})
	if err != nil {
		response.Diagnostics.AddWarning(
			"Incomplete Read",
			fmt.Sprintf("the elastic ip was attached but server %d could not be read back: %s", serverID, err),
		)
		diagnostics = response.State.Set(ctx, config)
		response.Diagnostics.Append(diagnostics...)
		return
	}

	var state computeElasticIPServerAttachmentResourceData
	state.FromEntity(server, elasticIP)

	state.NetworkInterfaceID = config.NetworkInterfaceID

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeElasticIPServerAttachmentResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeElasticIPServerAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	server, err := c.client.Compute.Server.Get(ctx, compute.ServerGetReq{ID: uint(state.ServerID.ValueInt64())})
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("server %d", state.ServerID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get server: %s", err))
		return
	}

	elasticIP, found, err := findComputeElasticIP(ctx, c.client.Compute.ElasticIP, int(state.ElasticIPID.ValueInt64()))
	if err != nil {
		response.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if !found {
		removeGone(ctx, response, fmt.Sprintf("elastic ip %d", state.ElasticIPID.ValueInt64()))
		return
	}

	state.FromEntity(server, elasticIP)

	if state.NetworkInterfaceID.IsNull() {
		removeGone(ctx, response, fmt.Sprintf("attachment of elastic ip %d to server %d", state.ElasticIPID.ValueInt64(), state.ServerID.ValueInt64()))
		return
	}

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeElasticIPServerAttachmentResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	response.Diagnostics.AddError("Not Supported", "updating an elastic ip attachment is not supported")
}

func (c computeElasticIPServerAttachmentResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeElasticIPServerAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "detach elastic ip", func() error {
		return c.client.Compute.ElasticIPAttachment.Delete(ctx, compute.ElasticIPAttachmentDeleteReq{
			ServerID:    uint(state.ServerID.ValueInt64()),
			ElasticIPID: uint(state.ElasticIPID.ValueInt64()),
		})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to detach elastic ip: %s", err))
		return
	}
}

func (c computeElasticIPServerAttachmentResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStateCompositeInt64IDs(ctx, request, response, path.Root("server_id"), path.Root("elastic_ip_id"))
}
