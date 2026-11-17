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
	_ datasource.DataSource              = (*computeSecurityGroupDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeSecurityGroupDataSource)(nil)
)

type computeSecurityGroupDataSourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
}

func (c *computeSecurityGroupDataSourceData) FromEntity(securityGroup compute.SecurityGroup) {
	c.ID = types.Int64Value(int64(securityGroup.ID))
	c.Name = types.StringValue(securityGroup.Name)
	c.LocationID = types.Int64Value(int64(securityGroup.Location.ID))
}

func (c computeSecurityGroupDataSourceData) AppliesTo(securityGroup compute.SecurityGroup) bool {
	if !c.ID.IsNull() && securityGroup.ID != int(c.ID.ValueInt64()) {
		return false
	}

	if !c.Name.IsNull() && securityGroup.Name != c.Name.ValueString() {
		return false
	}

	if !c.LocationID.IsNull() && securityGroup.Location.ID != int(c.LocationID.ValueInt64()) {
		return false
	}

	return true
}

func (c computeSecurityGroupDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the security group",
				Optional:            true,
				Computed:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func newComputeSecurityGroupDataSource() datasource.DataSource {
	return &computeSecurityGroupDataSource{}
}

func (c *computeSecurityGroupDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_security_group"
}

func (c *computeSecurityGroupDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeSecurityGroupDataSource struct {
	client flowClient
}

func (c computeSecurityGroupDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeSecurityGroupDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.client.Compute.SecurityGroup.List(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list security groups: %s", err))
		return
	}

	securityGroup, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find security group: %s", err))
		return
	}

	var state computeSecurityGroupDataSourceData
	state.FromEntity(securityGroup)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
