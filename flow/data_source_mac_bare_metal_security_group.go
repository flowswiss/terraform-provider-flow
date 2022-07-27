package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ tfsdk.DataSourceType = (*macBareMetalSecurityGroupDataSourceType)(nil)
	_ tfsdk.DataSource     = (*macBareMetalSecurityGroupDataSource)(nil)
)

type macBareMetalSecurityGroupDataSourceData struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	NetworkID types.Int64  `tfsdk:"network_id"`
}

func (c *macBareMetalSecurityGroupDataSourceData) FromEntity(securityGroup macbaremetal.SecurityGroup) {
	c.ID = types.Int64{Value: int64(securityGroup.ID)}
	c.Name = types.String{Value: securityGroup.Name}
	c.NetworkID = types.Int64{Value: int64(securityGroup.Network.ID)}
}

func (c macBareMetalSecurityGroupDataSourceData) AppliesTo(securityGroup macbaremetal.SecurityGroup) bool {
	if !c.ID.Null && securityGroup.ID != int(c.ID.Value) {
		return false
	}

	if !c.Name.Null && securityGroup.Name != c.Name.Value {
		return false
	}

	if !c.NetworkID.Null && securityGroup.Network.ID != int(c.NetworkID.Value) {
		return false
	}

	return true
}

type macBareMetalSecurityGroupDataSourceType struct{}

func (c macBareMetalSecurityGroupDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
			"network_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network",
				Optional:            true,
				Computed:            true,
			},
		},
	}, nil
}

func (c macBareMetalSecurityGroupDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return macBareMetalSecurityGroupDataSource{
		securityGroupService: macbaremetal.NewSecurityGroupService(prov.client),
	}, diagnostics
}

type macBareMetalSecurityGroupDataSource struct {
	securityGroupService macbaremetal.SecurityGroupService
}

func (c macBareMetalSecurityGroupDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config macBareMetalSecurityGroupDataSourceData
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

	var state macBareMetalSecurityGroupDataSourceData
	state.FromEntity(securityGroup)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
