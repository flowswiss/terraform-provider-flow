package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/common"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var _ datasource.DataSource = (*locationDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*locationDataSource)(nil)

type locationDataSourceData struct {
	ID               types.Int64            `tfsdk:"id"`
	Name             types.String           `tfsdk:"name"`
	Key              types.String           `tfsdk:"key"`
	RequiredModules  []moduleDataSourceData `tfsdk:"required_modules"`
	AvailableModules []moduleDataSourceData `tfsdk:"available_modules"`
}

func (l *locationDataSourceData) FromEntity(location common.Location) {
	l.ID = types.Int64Value(int64(location.ID))
	l.Name = types.StringValue(location.Name)
	l.Key = types.StringValue(location.Key)

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
	if !l.ID.IsNull() && location.ID != int(l.ID.ValueInt64()) {
		return false
	}

	if !l.Name.IsNull() && location.Name != l.Name.ValueString() {
		return false
	}

	if !l.Key.IsNull() && location.Key != l.Key.ValueString() {
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

func (l locationDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	var moduleSchema datasource.SchemaResponse
	moduleDataSource{}.Schema(ctx, datasource.SchemaRequest{}, &moduleSchema)

	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the location",
				Optional:            true,
				Computed:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "key of the location",
				Optional:            true,
				Computed:            true,
			},
			"required_modules": schema.ListNestedAttribute{
				NestedObject:        schema.NestedAttributeObject{Attributes: moduleSchema.Schema.Attributes},
				MarkdownDescription: "list of required modules",
				Optional:            true,
			},
			"available_modules": schema.ListNestedAttribute{
				NestedObject:        schema.NestedAttributeObject{Attributes: moduleSchema.Schema.Attributes},
				MarkdownDescription: "list of available modules",
				Computed:            true,
			},
		},
	}
}

func newLocationDataSource() datasource.DataSource {
	return &locationDataSource{}
}

func (l *locationDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_location"
}

func (l *locationDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	l.client = client
}

type locationDataSource struct {
	client flowClient
}

func (l locationDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config locationDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := l.client.Common.Location.List(ctx, core.CursorAll)
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
