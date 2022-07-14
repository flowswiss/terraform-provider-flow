package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var _ tfsdk.DataSourceType = (*moduleDataSourceType)(nil)
var _ tfsdk.DataSource = (*moduleDataSource)(nil)

type moduleDataSourceData struct {
	ID     types.Int64  `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Parent types.Object `tfsdk:"parent"`
}

func (m *moduleDataSourceData) FromEntity(module common.Module) {
	m.ID = types.Int64{Value: int64(module.ID)}
	m.Name = types.String{Value: module.Name}

	m.Parent = types.Object{
		AttrTypes: map[string]attr.Type{
			"id":   types.Int64Type,
			"name": types.StringType,
		},
	}

	if module.Parent == nil {
		m.Parent.Null = true
	} else {
		m.Parent.Attrs = map[string]attr.Value{
			"id":   types.Int64{Value: int64(module.Parent.ID)},
			"name": types.String{Value: module.Parent.Name},
		}
	}
}

func (m moduleDataSourceData) AppliesTo(module common.Module) bool {
	if !m.ID.Null && m.ID.Value != int64(module.ID) {
		return false
	}

	if !m.Name.Null && m.Name.Value != module.Name {
		return false
	}

	return true
}

type moduleDataSourceType struct{}

func (l moduleDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the module",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the module",
				Optional:            true,
				Computed:            true,
			},
			"parent": {
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"id": {
						Type:                types.Int64Type,
						MarkdownDescription: "unique identifier of the parent module",
						Computed:            true,
					},
					"name": {
						Type:                types.StringType,
						MarkdownDescription: "name of the parent module",
						Computed:            true,
					},
				}),
				MarkdownDescription: "parent module",
				Computed:            true,
			},
		},
	}, nil
}

func (l moduleDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return moduleDataSource{
		client: prov.client,
	}, diagnostics
}

type moduleDataSource struct {
	client goclient.Client
}

func (l moduleDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config moduleDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := common.NewModuleService(l.client).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get modules: %s", err))
		return
	}

	module, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find module: %s", err))
		return
	}

	var state moduleDataSourceData
	state.FromEntity(module)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
