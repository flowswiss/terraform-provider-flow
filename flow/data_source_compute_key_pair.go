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
	_ tfsdk.DataSourceType = (*computeKeyPairDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeKeyPairDataSource)(nil)
)

type computeKeyPairDataSourceData struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Fingerprint types.String `tfsdk:"fingerprint"`
}

func (c *computeKeyPairDataSourceData) FromEntity(keyPair compute.KeyPair) {
	c.ID = types.Int64{Value: int64(keyPair.ID)}
	c.Name = types.String{Value: keyPair.Name}
	c.Fingerprint = types.String{Value: keyPair.Fingerprint}
}

func (c computeKeyPairDataSourceData) AppliesTo(keyPair compute.KeyPair) bool {
	if !c.ID.Null && int(c.ID.Value) != keyPair.ID {
		return false
	}

	if !c.Name.Null && c.Name.Value != keyPair.Name {
		return false
	}

	if !c.Fingerprint.Null && c.Fingerprint.Value != keyPair.Fingerprint {
		return false
	}

	return true
}

type computeKeyPairDataSourceType struct{}

func (t computeKeyPairDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the key pair",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the key pair",
				Optional:            true,
				Computed:            true,
			},
			"fingerprint": {
				Type:                types.StringType,
				MarkdownDescription: "fingerprint of the key pair",
				Optional:            true,
				Computed:            true,
			},
		},
	}, nil
}

func (t computeKeyPairDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeKeyPairDataSource{
		keyPairService: compute.NewKeyPairService(prov.client),
	}, diagnostics
}

type computeKeyPairDataSource struct {
	keyPairService compute.KeyPairService
}

func (s computeKeyPairDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeKeyPairDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := s.keyPairService.List(ctx, goclient.Cursor{NoFilter: 1})
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
