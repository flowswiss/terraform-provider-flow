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
	_ tfsdk.DataSourceType = (*computeSecurityGroupDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeSecurityGroupDataSource)(nil)
)

type computeSecurityGroupDataSourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
}

func (c *computeSecurityGroupDataSourceData) FromEntity(securityGroup compute.SecurityGroup) {
	c.ID = types.Int64{Value: int64(securityGroup.ID)}
	c.Name = types.String{Value: securityGroup.Name}
	c.LocationID = types.Int64{Value: int64(securityGroup.Location.ID)}
}

func (c computeSecurityGroupDataSourceData) AppliesTo(securityGroup compute.SecurityGroup) bool {
	if !c.ID.Null && securityGroup.ID != int(c.ID.Value) {
		return false
	}

	if !c.Name.Null && securityGroup.Name != c.Name.Value {
		return false
	}

	if !c.LocationID.Null && securityGroup.Location.ID != int(c.LocationID.Value) {
		return false
	}

	return true
}

type computeSecurityGroupDataSourceType struct{}

func (c computeSecurityGroupDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the security group",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the security group",
				Optional:            true,
				Computed:            true,
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the location",
				Optional:            true,
				Computed:            true,
			},
		},
	}, nil
}

func (c computeSecurityGroupDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeSecurityGroupDataSource{
		securityGroupService: compute.NewSecurityGroupService(prov.client),
	}, diagnostics
}

type computeSecurityGroupDataSource struct {
	securityGroupService compute.SecurityGroupService
}

func (c computeSecurityGroupDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeSecurityGroupDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.securityGroupService.List(ctx, goclient.Cursor{NoFilter: 1})
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
