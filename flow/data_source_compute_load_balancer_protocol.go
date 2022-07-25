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
	_ tfsdk.DataSourceType = (*computeLoadBalancerProtocolDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeLoadBalancerProtocolDataSource)(nil)
)

type computeLoadBalancerProtocolDataSourceData struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Key  types.String `tfsdk:"key"`
}

func (c *computeLoadBalancerProtocolDataSourceData) FromEntity(protocol compute.LoadBalancerProtocol) {
	c.ID = types.Int64{Value: int64(protocol.ID)}
	c.Name = types.String{Value: protocol.Name}
	c.Key = types.String{Value: protocol.Key}
}

func (c computeLoadBalancerProtocolDataSourceData) AppliesTo(protocol compute.LoadBalancerProtocol) bool {
	if !c.ID.Null && int(c.ID.Value) != protocol.ID {
		return false
	}

	if !c.Name.Null && c.Name.Value != protocol.Name {
		return false
	}

	if !c.Key.Null && c.Key.Value != protocol.Key {
		return false
	}

	return true
}

type computeLoadBalancerProtocolDataSourceType struct{}

func (c computeLoadBalancerProtocolDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the load balancer protocol",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the load balancer protocol",
				Optional:            true,
				Computed:            true,
			},
			"key": {
				Type:                types.StringType,
				MarkdownDescription: "unique key of the load balancer protocol",
				Optional:            true,
				Computed:            true,
			},
		},
	}, nil
}

func (c computeLoadBalancerProtocolDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeLoadBalancerProtocolDataSource{
		loadBalancerEntityService: compute.NewLoadBalancerEntityService(prov.client),
	}, diagnostics
}

type computeLoadBalancerProtocolDataSource struct {
	loadBalancerEntityService compute.LoadBalancerEntityService
}

func (c computeLoadBalancerProtocolDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeLoadBalancerProtocolDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.loadBalancerEntityService.ListProtocols(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list load balancer protocols: %s", err))
		return
	}

	protocol, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find load balancer protocol: %s", err))
		return
	}

	var state computeLoadBalancerProtocolDataSourceData
	state.FromEntity(protocol)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
