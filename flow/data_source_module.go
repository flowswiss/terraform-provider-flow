package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/common"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var _ datasource.DataSource = (*moduleDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*moduleDataSource)(nil)

type moduleDataSourceData struct {
	ID     types.Int64  `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Parent types.Object `tfsdk:"parent"`
}

func (m *moduleDataSourceData) FromEntity(module common.Module) {
	m.ID = types.Int64Value(int64(module.ID))
	m.Name = types.StringValue(module.Name)

	parentTypes := map[string]attr.Type{
		"id":   types.Int64Type,
		"name": types.StringType,
	}

	if module.Parent == nil {
		m.Parent = types.ObjectNull(parentTypes)
	} else {
		m.Parent = types.ObjectValueMust(parentTypes, map[string]attr.Value{
			"id":   types.Int64Value(int64(module.Parent.ID)),
			"name": types.StringValue(module.Parent.Name),
		})
	}
}

func (m moduleDataSourceData) AppliesTo(module common.Module) bool {
	if !m.ID.IsNull() && m.ID.ValueInt64() != int64(module.ID) {
		return false
	}

	if !m.Name.IsNull() && m.Name.ValueString() != module.Name {
		return false
	}

	return true
}

func (l moduleDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the module",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the module",
				Optional:            true,
				Computed:            true,
			},
			"parent": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						MarkdownDescription: "unique identifier of the parent module",
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "name of the parent module",
						Computed:            true,
					},
				},
				MarkdownDescription: "parent module",
				Computed:            true,
			},
		},
	}
}

func newModuleDataSource() datasource.DataSource {
	return &moduleDataSource{}
}

func (l *moduleDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_module"
}

func (l *moduleDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	l.client = client
}

type moduleDataSource struct {
	client flowClient
}

func (l moduleDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config moduleDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := l.client.Common.Module.List(ctx, core.CursorAll)
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
