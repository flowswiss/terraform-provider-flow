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
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ tfsdk.ResourceType            = (*computeElasticIPResourceType)(nil)
	_ tfsdk.Resource                = (*computeElasticIPResource)(nil)
	_ tfsdk.ResourceWithImportState = (*computeElasticIPResource)(nil)
)

type computeElasticIPResourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	LocationID types.Int64  `tfsdk:"location_id"`
	PublicIP   types.String `tfsdk:"public_ip"`
}

func (c *computeElasticIPResourceData) FromEntity(elasticIP compute.ElasticIP) {
	c.ID = types.Int64{Value: int64(elasticIP.ID)}
	c.LocationID = types.Int64{Value: int64(elasticIP.Location.ID)}
	c.PublicIP = types.String{Value: elasticIP.PublicIP}
}

type computeElasticIPResourceType struct{}

func (c computeElasticIPResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the elastic ip",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "location of the elastic ip",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"public_ip": {
				Type:                types.StringType,
				MarkdownDescription: "public ip address",
				Computed:            true,
			},
		},
	}, nil
}

func (c computeElasticIPResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeElasticIPResource{
		elasticIPService: compute.NewElasticIPService(prov.client),
	}, diagnostics
}

type computeElasticIPResource struct {
	elasticIPService compute.ElasticIPService
}

func (c computeElasticIPResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeElasticIPResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.ElasticIPCreate{
		LocationID: int(config.LocationID.Value),
	}

	elasticIP, err := c.elasticIPService.Create(ctx, create)
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

func (c computeElasticIPResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeElasticIPResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	elasticIP, diagnostics := findElasticIP(ctx, c.elasticIPService, int(state.ID.Value))
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	state.FromEntity(elasticIP)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeElasticIPResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	response.Diagnostics.AddError("Not Supported", "updating an elastic ip is not supported")
}

func (c computeElasticIPResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeElasticIPResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := c.elasticIPService.Delete(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete elastic ip: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted elastic ip", map[string]interface{}{
		"id": int(state.ID.Value),
	})
}

func (c computeElasticIPResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, path.Root("id"), request, response)
}

func findElasticIP(ctx context.Context, service compute.ElasticIPService, id int) (elasticIP compute.ElasticIP, diagnostics diag.Diagnostics) {
	list, err := service.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("unable to list elastic ips: %s", err))
		return
	}

	for _, elasticIP = range list.Items {
		if elasticIP.ID == id {
			return
		}
	}

	diagnostics.AddError("Not Found", fmt.Sprintf("unable to find elastic ip with id %d", id))
	return
}
