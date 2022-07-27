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
	_ tfsdk.DataSourceType = (*macBareMetalElasticIPDataSourceType)(nil)
	_ tfsdk.DataSource     = (*macBareMetalElasticIPDataSource)(nil)
)

type macBareMetalElasticIPDataSourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	LocationID types.Int64  `tfsdk:"location_id"`
	PublicIP   types.String `tfsdk:"public_ip"`
}

func (c *macBareMetalElasticIPDataSourceData) FromEntity(elasticIP macbaremetal.ElasticIP) {
	c.ID = types.Int64{Value: int64(elasticIP.ID)}
	c.LocationID = types.Int64{Value: int64(elasticIP.Location.ID)}
	c.PublicIP = types.String{Value: elasticIP.PublicIP}
}

func (c macBareMetalElasticIPDataSourceData) AppliesTo(elasticIP macbaremetal.ElasticIP) bool {
	if !c.ID.Null && c.ID.Value != int64(elasticIP.ID) {
		return false
	}

	if !c.LocationID.Null && c.LocationID.Value != int64(elasticIP.Location.ID) {
		return false
	}

	if !c.PublicIP.Null && c.PublicIP.Value != elasticIP.PublicIP {
		return false
	}

	return true
}

type macBareMetalElasticIPDataSourceType struct{}

func (c macBareMetalElasticIPDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the elastic ip",
				Optional:            true,
				Computed:            true,
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "location of the elastic ip",
				Optional:            true,
				Computed:            true,
			},
			"public_ip": {
				Type:                types.StringType,
				MarkdownDescription: "public ip address",
				Optional:            true,
				Computed:            true,
			},
		},
	}, nil
}

func (c macBareMetalElasticIPDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return macBareMetalElasticIPDataSource{
		elasticIPService: macbaremetal.NewElasticIPService(prov.client),
	}, diagnostics
}

type macBareMetalElasticIPDataSource struct {
	elasticIPService macbaremetal.ElasticIPService
}

func (c macBareMetalElasticIPDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config macBareMetalElasticIPDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.elasticIPService.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list elastic ips: %s", err))
		return
	}

	elasticIP, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find elastic ip: %s", err))
		return
	}

	var state macBareMetalElasticIPDataSourceData
	state.FromEntity(elasticIP)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
