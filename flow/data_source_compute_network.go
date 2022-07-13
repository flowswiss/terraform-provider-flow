package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.DataSourceType = (*computeNetworkDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeNetworkDataSource)(nil)
)

type computeNetworkDataSourceAllocationPool struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

type computeNetworkDataSourceData struct {
	ID                types.Int64                             `tfsdk:"id"`
	Name              types.String                            `tfsdk:"name"`
	CIDR              types.String                            `tfsdk:"cidr"`
	LocationID        types.Int64                             `tfsdk:"location_id"`
	DomainNameServers []types.String                          `tfsdk:"domain_name_servers"`
	AllocationPool    *computeNetworkDataSourceAllocationPool `tfsdk:"allocation_pool"`
	GatewayIP         types.String                            `tfsdk:"gateway_ip"`
}

func (c *computeNetworkDataSourceData) FromEntity(network compute.Network) {
	c.ID = types.Int64{Value: int64(network.ID)}
	c.Name = types.String{Value: network.Name}
	c.CIDR = types.String{Value: network.CIDR}
	c.LocationID = types.Int64{Value: int64(network.Location.ID)}
	c.GatewayIP = types.String{Value: network.GatewayIP}

	c.AllocationPool = &computeNetworkDataSourceAllocationPool{
		Start: types.String{Value: network.AllocationPoolStart},
		End:   types.String{Value: network.AllocationPoolEnd},
	}

	c.DomainNameServers = make([]types.String, len(network.DomainNameServers))
	for idx, domainNameServer := range network.DomainNameServers {
		c.DomainNameServers[idx] = types.String{Value: domainNameServer}
	}
}

type computeNetworkDataSourceType struct{}

func (c computeNetworkDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the network",
				Optional:            true,
				Computed:            true,
			},
			"cidr": {
				Type:                types.StringType,
				MarkdownDescription: "CIDR of the network",
				Computed:            true,
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the location",
				Computed:            true,
			},
			"domain_name_servers": {
				Type: types.ListType{
					ElemType: types.StringType,
				},
				MarkdownDescription: "list of domain name servers",
				Computed:            true,
			},
			"allocation_pool": {
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"start": {
						Type:                types.StringType,
						MarkdownDescription: "start of the allocation pool",
						Computed:            true,
					},
					"end": {
						Type:                types.StringType,
						MarkdownDescription: "end of the allocation pool",
						Computed:            true,
					},
				}),
				MarkdownDescription: "allocation pool",
				Computed:            true,
			},
			"gateway_ip": {
				Type:                types.StringType,
				MarkdownDescription: "gateway IP of the network",
				Computed:            true,
			},
		},
	}, nil
}

func (c computeNetworkDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeNetworkDataSource{
		networkService: compute.NewNetworkService(prov.client),
	}, diagnostics
}

type computeNetworkDataSource struct {
	networkService compute.NetworkService
}

func (c computeNetworkDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeNetworkDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	items, err := c.networkService.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list networks: %s", err))
		return
	}

	for _, network := range items.Items {
		if !config.ID.Null && network.ID != int(config.ID.Value) {
			continue
		}

		if !config.Name.Null && network.Name != config.Name.Value {
			continue
		}

		var state computeNetworkDataSourceData
		state.FromEntity(network)

		diagnostics = response.State.Set(ctx, state)
		response.Diagnostics.Append(diagnostics...)
		return
	}

	response.Diagnostics.AddError("Not Found", "requested network could not be found")
}
