package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*computeElasticIPResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeElasticIPResource)(nil)
	_ resource.ResourceWithImportState = (*computeElasticIPResource)(nil)
)

type computeElasticIPResourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	LocationID types.Int64  `tfsdk:"location_id"`
	PublicIP   types.String `tfsdk:"public_ip"`
}

func (c *computeElasticIPResourceData) FromEntity(elasticIP compute.ElasticIP) {
	c.ID = types.Int64Value(int64(elasticIP.ID))
	c.LocationID = types.Int64Value(int64(elasticIP.Location.ID))
	c.PublicIP = types.StringValue(elasticIP.PublicIP)
}

func (c computeElasticIPResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the elastic ip",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "location of the elastic ip",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"public_ip": schema.StringAttribute{
				MarkdownDescription: "public ip address",
				Computed:            true,
			},
		},
	}
}

func newComputeElasticIPResource() resource.Resource {
	return &computeElasticIPResource{}
}

func (c *computeElasticIPResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_elastic_ip"
}

func (c *computeElasticIPResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeElasticIPResource struct {
	client flowClient
}

func (c computeElasticIPResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeElasticIPResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.ElasticIPCreateReq{
		LocationID: int(config.LocationID.ValueInt64()),
	}

	var elasticIP compute.ElasticIP
	err := retryCreate(ctx, "create elastic ip", func() (err error) {
		elasticIP, err = c.client.Compute.ElasticIP.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create elastic ip: %s", err))
		return
	}

	tflog.Trace(ctx, "created elastic ip", map[string]interface{}{
		"id":   elasticIP.ID,
		"data": elasticIP,
	})

	var state computeElasticIPResourceData
	state.FromEntity(elasticIP)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeElasticIPResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeElasticIPResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	elasticIP, found, err := findComputeElasticIP(ctx, c.client.Compute.ElasticIP, int(state.ID.ValueInt64()))
	if err != nil {
		response.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if !found {
		removeGone(ctx, response, fmt.Sprintf("elastic ip %d", state.ID.ValueInt64()))
		return
	}

	state.FromEntity(elasticIP)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeElasticIPResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	response.Diagnostics.AddError("Not Supported", "updating an elastic ip is not supported")
}

func (c computeElasticIPResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeElasticIPResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "delete elastic ip", func() error {
		return c.client.Compute.ElasticIP.Delete(ctx, compute.ElasticIPDeleteReq{ID: int(state.ID.ValueInt64())})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete elastic ip: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted elastic ip", map[string]interface{}{
		"id": int(state.ID.ValueInt64()),
	})
}

func (c computeElasticIPResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}

func findComputeElasticIP(ctx context.Context, service *compute.ElasticIPService, id int) (elasticIP compute.ElasticIP, found bool, err error) {
	list, err := service.List(ctx, core.CursorAll)
	if err != nil {
		return elasticIP, false, fmt.Errorf("unable to list elastic ips: %w", err)
	}

	for _, elasticIP = range list.Items {
		if elasticIP.ID == id {
			return elasticIP, true, nil
		}
	}

	return compute.ElasticIP{}, false, nil
}
