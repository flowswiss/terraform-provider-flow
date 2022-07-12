package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ tfsdk.DataSourceType = (*locationDataSourceType)(nil)
var _ tfsdk.DataSource = (*locationDataSource)(nil)

type locationDataSourceData struct {
	ID              types.Int64  `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Key             types.String `tfsdk:"key"`
	RequiredModules types.List   `tfsdk:"required_modules"`
}

func (l *locationDataSourceData) FromEntity(location common.Location) {
	l.ID = types.Int64{Value: int64(location.ID)}
	l.Name = types.String{Value: location.Name}
	l.Key = types.String{Value: location.Key}
}

type locationDataSourceType struct{}

func (l locationDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the location",
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the location",
				Optional:            true,
			},
			"key": {
				Type:                types.StringType,
				MarkdownDescription: "key of the location",
				Optional:            true,
			},
			"required_modules": {
				Type:                types.ListType{ElemType: types.Int64Type},
				MarkdownDescription: "list of required modules",
				Optional:            true,
			},
		},
	}, nil
}

func (l locationDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return &locationDataSource{
		client: prov.client,
	}, diagnostics
}

type locationDataSource struct {
	client goclient.Client
}

func (l locationDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config locationDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := common.NewLocationService(l.client).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get locations: %s", err))
		return
	}

	var state locationDataSourceData

	for _, location := range list.Items {
		if !config.Name.Null && location.Name != config.Name.Value {
			continue
		}

		if !config.Key.Null && location.Key != config.Key.Value {
			continue
		}

		if !config.RequiredModules.Null {
			hasRequiredModules := true
			for _, requiredModule := range config.RequiredModules.Elems {
				if !locationHasModule(location, int(requiredModule.(types.Int64).Value)) {
					hasRequiredModules = false
					break
				}
			}

			if !hasRequiredModules {
				continue
			}
		}

		state.FromEntity(location)
		state.RequiredModules = config.RequiredModules

		diagnostics = response.State.Set(ctx, state)
		response.Diagnostics.Append(diagnostics...)
		return
	}

	response.Diagnostics.AddError("Not Found", "no location found matching the given criteria")
}

func locationHasModule(location common.Location, requiredModule int) bool {
	for _, availableModule := range location.Modules {
		if requiredModule == availableModule.ID {
			return true
		}
	}

	return false
}
