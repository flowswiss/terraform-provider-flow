package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ datasource.DataSource              = (*computeNetworkDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeNetworkDataSource)(nil)
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
	c.ID = types.Int64Value(int64(network.ID))
	c.Name = types.StringValue(network.Name)
	c.CIDR = types.StringValue(network.CIDR)
	c.LocationID = types.Int64Value(int64(network.Location.ID))
	c.GatewayIP = types.StringValue(network.GatewayIP)

	c.AllocationPool = &computeNetworkDataSourceAllocationPool{
		Start: types.StringValue(network.AllocationPoolStart),
		End:   types.StringValue(network.AllocationPoolEnd),
	}

	c.DomainNameServers = make([]types.String, len(network.DomainNameServers))
	for idx, domainNameServer := range network.DomainNameServers {
		c.DomainNameServers[idx] = types.StringValue(domainNameServer)
	}
}

func (c computeNetworkDataSourceData) AppliesTo(network compute.Network) bool {
	if !c.ID.IsNull() && network.ID != int(c.ID.ValueInt64()) {
		return false
	}

	if !c.Name.IsNull() && network.Name != c.Name.ValueString() {
		return false
	}

	return true
}

func (c computeNetworkDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the network",
				Optional:            true,
				Computed:            true,
			},
			"cidr": schema.StringAttribute{
				MarkdownDescription: "CIDR of the network",
				Computed:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Computed:            true,
			},
			"domain_name_servers": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "list of domain name servers",
				Computed:            true,
			},
			"allocation_pool": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"start": schema.StringAttribute{
						MarkdownDescription: "start of the allocation pool",
						Computed:            true,
					},
					"end": schema.StringAttribute{
						MarkdownDescription: "end of the allocation pool",
						Computed:            true,
					},
				},
				MarkdownDescription: "allocation pool",
				Computed:            true,
			},
			"gateway_ip": schema.StringAttribute{
				MarkdownDescription: "gateway IP of the network",
				Computed:            true,
			},
		},
	}
}

func newComputeNetworkDataSource() datasource.DataSource {
	return &computeNetworkDataSource{}
}

func (c *computeNetworkDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_network"
}

func (c *computeNetworkDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeNetworkDataSource struct {
	client flowClient
}

func (c computeNetworkDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeNetworkDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.client.Compute.Network.List(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list networks: %s", err))
		return
	}

	network, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find network: %s", err))
		return
	}

	// the list endpoint leaves out the name servers, the allocation pool and the gateway
	network, err = c.client.Compute.Network.Get(ctx, compute.NetworkGetReq{ID: uint(network.ID)})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get network: %s", err))
		return
	}

	var state computeNetworkDataSourceData
	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
