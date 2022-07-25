package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ tfsdk.DataSourceType = (*computeLoadBalancerHealthCheckTypeDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeLoadBalancerHealthCheckTypeDataSource)(nil)
)

type computeLoadBalancerHealthCheckTypeDataSourceData struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Key  types.String `tfsdk:"key"`
}

func (c *computeLoadBalancerHealthCheckTypeDataSourceData) FromEntity(healthCheckType compute.LoadBalancerHealthCheckType) {
	c.ID = types.Int64{Value: int64(healthCheckType.ID)}
	c.Name = types.String{Value: healthCheckType.Name}
	c.Key = types.String{Value: healthCheckType.Key}
}

func (c computeLoadBalancerHealthCheckTypeDataSourceData) AppliesTo(healthCheckType compute.LoadBalancerHealthCheckType) bool {
	if !c.ID.Null && int(c.ID.Value) != healthCheckType.ID {
		return false
	}

	if !c.Name.Null && c.Name.Value != healthCheckType.Name {
		return false
	}

	if !c.Key.Null && c.Key.Value != healthCheckType.Key {
		return false
	}

	return true
}

type computeLoadBalancerHealthCheckTypeDataSourceType struct{}

func (c computeLoadBalancerHealthCheckTypeDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the load balancer health check type",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the load balancer health check type",
				Optional:            true,
				Computed:            true,
			},
			"key": {
				Type:                types.StringType,
				MarkdownDescription: "unique key of the load balancer health check type",
				Optional:            true,
				Computed:            true,
			},
		},
	}, nil
}

func (c computeLoadBalancerHealthCheckTypeDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeLoadBalancerHealthCheckTypeDataSource{
		loadBalancerEntityService: compute.NewLoadBalancerEntityService(prov.client),
	}, diagnostics
}

type computeLoadBalancerHealthCheckTypeDataSource struct {
	loadBalancerEntityService compute.LoadBalancerEntityService
}

func (c computeLoadBalancerHealthCheckTypeDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeLoadBalancerHealthCheckTypeDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.loadBalancerEntityService.ListHealthCheckTypes(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list load balancer health check types: %s", err))
		return
	}

	healthCheckType, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find load balancer health check type: %s", err))
		return
	}

	var state computeLoadBalancerHealthCheckTypeDataSourceData
	state.FromEntity(healthCheckType)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
