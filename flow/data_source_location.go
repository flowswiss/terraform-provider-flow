package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var _ tfsdk.DataSourceType = (*locationDataSourceType)(nil)
var _ tfsdk.DataSource = (*locationDataSource)(nil)

type locationDataSourceData struct {
	ID               types.Int64            `tfsdk:"id"`
	Name             types.String           `tfsdk:"name"`
	Key              types.String           `tfsdk:"key"`
	RequiredModules  []moduleDataSourceData `tfsdk:"required_modules"`
	AvailableModules []moduleDataSourceData `tfsdk:"available_modules"`
}

func (l *locationDataSourceData) FromEntity(location common.Location) {
	l.ID = types.Int64{Value: int64(location.ID)}
	l.Name = types.String{Value: location.Name}
	l.Key = types.String{Value: location.Key}

	if len(location.Modules) == 0 {
		l.AvailableModules = nil
	} else {
		l.AvailableModules = make([]moduleDataSourceData, len(location.Modules))
		for i, availableModule := range location.Modules {
			l.AvailableModules[i].FromEntity(availableModule)
		}
	}
}

func (l locationDataSourceData) AppliesTo(location common.Location) bool {
	if !l.ID.Null && location.ID != int(l.ID.Value) {
		return false
	}

	if !l.Name.Null && location.Name != l.Name.Value {
		return false
	}

	if !l.Key.Null && location.Key != l.Key.Value {
		return false
	}

	if len(l.RequiredModules) != 0 {
		for _, requiredModule := range l.RequiredModules {
			if modules := filter.Find(requiredModule, location.Modules); len(modules) == 0 {
				return false
			}
		}
	}

	return true
}

type locationDataSourceType struct{}

func (l locationDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	moduleSchema, diagnostics := moduleDataSourceType{}.GetSchema(ctx)
	if diagnostics.HasError() {
		return tfsdk.Schema{}, diagnostics
	}

	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the location",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the location",
				Optional:            true,
				Computed:            true,
			},
			"key": {
				Type:                types.StringType,
				MarkdownDescription: "key of the location",
				Optional:            true,
				Computed:            true,
			},
			"required_modules": {
				Attributes:          tfsdk.ListNestedAttributes(moduleSchema.Attributes),
				MarkdownDescription: "list of required modules",
				Optional:            true,
			},
			"available_modules": {
				Attributes:          tfsdk.ListNestedAttributes(moduleSchema.Attributes),
				MarkdownDescription: "list of available modules",
				Computed:            true,
			},
		},
	}, nil
}

func (l locationDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return locationDataSource{
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

	location, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find location: %s", err))
		return
	}

	var state locationDataSourceData
	state.FromEntity(location)
	state.RequiredModules = config.RequiredModules

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
