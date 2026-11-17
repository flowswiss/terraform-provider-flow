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
	_ datasource.DataSource              = (*computeKeyPairDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeKeyPairDataSource)(nil)
)

type computeKeyPairDataSourceData struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Fingerprint types.String `tfsdk:"fingerprint"`
}

func (c *computeKeyPairDataSourceData) FromEntity(keyPair compute.KeyPair) {
	c.ID = types.Int64Value(int64(keyPair.ID))
	c.Name = types.StringValue(keyPair.Name)
	c.Fingerprint = types.StringValue(keyPair.Fingerprint)
}

func (c computeKeyPairDataSourceData) AppliesTo(keyPair compute.KeyPair) bool {
	if !c.ID.IsNull() && int(c.ID.ValueInt64()) != keyPair.ID {
		return false
	}

	if !c.Name.IsNull() && c.Name.ValueString() != keyPair.Name {
		return false
	}

	if !c.Fingerprint.IsNull() && c.Fingerprint.ValueString() != keyPair.Fingerprint {
		return false
	}

	return true
}

func (t computeKeyPairDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the key pair",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the key pair",
				Optional:            true,
				Computed:            true,
			},
			"fingerprint": schema.StringAttribute{
				MarkdownDescription: "fingerprint of the key pair",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func newComputeKeyPairDataSource() datasource.DataSource {
	return &computeKeyPairDataSource{}
}

func (s *computeKeyPairDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_key_pair"
}

func (s *computeKeyPairDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	s.client = client
}

type computeKeyPairDataSource struct {
	client flowClient
}

func (s computeKeyPairDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeKeyPairDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := s.client.Compute.KeyPair.List(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list key pairs: %s", err))
		return
	}

	keyPair, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find key pair: %s", err))
		return
	}

	var state computeKeyPairDataSourceData
	state.FromEntity(keyPair)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
