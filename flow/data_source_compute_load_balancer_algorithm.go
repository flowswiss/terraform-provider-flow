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
	_ tfsdk.DataSourceType = (*computeLoadBalancerAlgorithmDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeLoadBalancerAlgorithmDataSource)(nil)
)

type computeLoadBalancerAlgorithmDataSourceData struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Key  types.String `tfsdk:"key"`
}

func (c *computeLoadBalancerAlgorithmDataSourceData) FromEntity(algorithm compute.LoadBalancerAlgorithm) {
	c.ID = types.Int64{Value: int64(algorithm.ID)}
	c.Name = types.String{Value: algorithm.Name}
	c.Key = types.String{Value: algorithm.Key}
}

func (c computeLoadBalancerAlgorithmDataSourceData) AppliesTo(algorithm compute.LoadBalancerAlgorithm) bool {
	if !c.ID.Null && int(c.ID.Value) != algorithm.ID {
		return false
	}

	if !c.Name.Null && c.Name.Value != algorithm.Name {
		return false
	}

	if !c.Key.Null && c.Key.Value != algorithm.Key {
		return false
	}

	return true
}

type computeLoadBalancerAlgorithmDataSourceType struct{}

func (c computeLoadBalancerAlgorithmDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the load balancer algorithm",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the load balancer algorithm",
				Optional:            true,
				Computed:            true,
			},
			"key": {
				Type:                types.StringType,
				MarkdownDescription: "unique key of the load balancer algorithm",
				Optional:            true,
				Computed:            true,
			},
		},
	}, nil
}

func (c computeLoadBalancerAlgorithmDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeLoadBalancerAlgorithmDataSource{
		loadBalancerEntityService: compute.NewLoadBalancerEntityService(prov.client),
	}, diagnostics
}

type computeLoadBalancerAlgorithmDataSource struct {
	loadBalancerEntityService compute.LoadBalancerEntityService
}

func (c computeLoadBalancerAlgorithmDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeLoadBalancerAlgorithmDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.loadBalancerEntityService.ListAlgorithms(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list load balancer algorithms: %s", err))
		return
	}

	algorithm, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find load balancer algorithm: %s", err))
		return
	}

	var state computeLoadBalancerAlgorithmDataSourceData
	state.FromEntity(algorithm)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
