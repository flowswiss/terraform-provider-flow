package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.ResourceType            = (*computeElasticIPServerAttachmentResourceType)(nil)
	_ tfsdk.Resource                = (*computeElasticIPServerAttachmentResource)(nil)
	_ tfsdk.ResourceWithImportState = (*computeElasticIPServerAttachmentResource)(nil)
)

type computeElasticIPServerAttachmentResourceData struct {
	ServerID           types.Int64 `tfsdk:"server_id"`
	NetworkInterfaceID types.Int64 `tfsdk:"network_interface_id"`
	ElasticIPID        types.Int64 `tfsdk:"elastic_ip_id"`
}

func (c *computeElasticIPServerAttachmentResourceData) FromEntity(server compute.Server, elasticIP compute.ElasticIP) {
	c.ServerID = types.Int64{Value: int64(server.ID)}
	c.NetworkInterfaceID = types.Int64{Null: true}
	c.ElasticIPID = types.Int64{Value: int64(elasticIP.ID)}

	for _, network := range server.Networks {
		for _, iface := range network.Interfaces {
			if iface.PublicIP == elasticIP.PublicIP {
				c.NetworkInterfaceID = types.Int64{Value: int64(iface.ID)}
			}
		}
	}
}

type computeElasticIPServerAttachmentResourceType struct{}

func (c computeElasticIPServerAttachmentResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Import: `terraform import flow_compute_elastic_ip_server_attachment.<name> <server_id>:<elastic_ip_id>`",
		Attributes: map[string]tfsdk.Attribute{
			"server_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the server to attach the elastic ip to",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"network_interface_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network interface to attach the elastic ip to — `flow_compute_server.<name>.network_interface_id` for a server's primary interface",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"elastic_ip_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the elastic ip to attach to the server",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (c computeElasticIPServerAttachmentResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeElasticIPServerAttachmentResource{
		serverService:    compute.NewServerService(prov.client),
		elasticIPService: compute.NewElasticIPService(prov.client),
		client:           prov.client,
	}, diagnostics
}

type computeElasticIPServerAttachmentResource struct {
	serverService    compute.ServerService
	elasticIPService compute.ElasticIPService

	client goclient.Client
}

func (c computeElasticIPServerAttachmentResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeElasticIPServerAttachmentResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	serverID := int(config.ServerID.Value)

	attach := compute.ElasticIPAttach{
		ElasticIPID:        int(config.ElasticIPID.Value),
		NetworkInterfaceID: int(config.NetworkInterfaceID.Value),
	}

	var elasticIP compute.ElasticIP
	err := retry(ctx, "attach elastic ip", func() (err error) {
		elasticIP, err = compute.NewServerElasticIPService(c.client, serverID).Attach(ctx, attach)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to attach elastic ip: %s", err))
		return
	}

	server, err := compute.NewServerService(c.client).Get(ctx, serverID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get server: %s", err))
		return
	}

	var state computeElasticIPServerAttachmentResourceData
	state.FromEntity(server, elasticIP)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeElasticIPServerAttachmentResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeElasticIPServerAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	server, err := compute.NewServerService(c.client).Get(ctx, int(state.ServerID.Value))
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("server %d", state.ServerID.Value))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get server: %s", err))
		return
	}

	elasticIP, found, err := findComputeElasticIP(ctx, c.elasticIPService, int(state.ElasticIPID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if !found {
		removeGone(ctx, response, fmt.Sprintf("elastic ip %d", state.ElasticIPID.Value))
		return
	}

	state.FromEntity(server, elasticIP)

	// no interface of the server carries the ip any more — detached outside terraform
	if state.NetworkInterfaceID.Null {
		removeGone(ctx, response, fmt.Sprintf("attachment of elastic ip %d to server %d", state.ElasticIPID.Value, state.ServerID.Value))
		return
	}

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeElasticIPServerAttachmentResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	response.Diagnostics.AddError("Not Supported", "updating an elastic ip attachment is not supported")
}

func (c computeElasticIPServerAttachmentResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeElasticIPServerAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "detach elastic ip", func() error {
		return compute.NewServerElasticIPService(c.client, int(state.ServerID.Value)).Detach(ctx, int(state.ElasticIPID.Value))
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to detach elastic ip: %s", err))
		return
	}
}

func (c computeElasticIPServerAttachmentResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	importStateCompositeInt64IDs(ctx, request, response, path.Root("server_id"), path.Root("elastic_ip_id"))
}
