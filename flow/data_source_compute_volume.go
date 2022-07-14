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
	_ tfsdk.DataSourceType = (*computeVolumeDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeVolumeDataSource)(nil)
)

type computeVolumeDataSourceData struct {
	ID           types.Int64  `tfsdk:"id"`
	SerialNumber types.String `tfsdk:"serial_number"`
	Name         types.String `tfsdk:"name"`
	Size         types.Int64  `tfsdk:"size"`
	LocationID   types.Int64  `tfsdk:"location_id"`
}

func (c *computeVolumeDataSourceData) FromEntity(volume compute.Volume) {
	c.ID = types.Int64{Value: int64(volume.ID)}
	c.SerialNumber = types.String{Value: volume.SerialNumber}
	c.Name = types.String{Value: volume.Name}
	c.Size = types.Int64{Value: int64(volume.Size)}
	c.LocationID = types.Int64{Value: int64(volume.Location.ID)}
}

func (c computeVolumeDataSourceData) AppliesTo(volume compute.Volume) bool {
	if !c.ID.Null && c.ID.Value != int64(volume.ID) {
		return false
	}

	if !c.SerialNumber.Null && c.SerialNumber.Value != volume.SerialNumber {
		return false
	}

	if !c.Name.Null && c.Name.Value != volume.Name {
		return false
	}

	if !c.LocationID.Null && c.LocationID.Value != int64(volume.Location.ID) {
		return false
	}

	return true
}

type computeVolumeDataSourceType struct{}

func (c computeVolumeDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the volume",
				Optional:            true,
				Computed:            true,
			},
			"serial_number": {
				Type:                types.StringType,
				MarkdownDescription: "unique serial number of the volume",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the volume",
				Optional:            true,
				Computed:            true,
			},
			"size": {
				Type:                types.Int64Type,
				MarkdownDescription: "size in GiB of the volume",
				Computed:            true,
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "identifier of the location of the volume",
				Optional:            true,
				Computed:            true,
			},
		},
	}, nil
}

func (c computeVolumeDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeVolumeDataSource{
		volumeService: compute.NewVolumeService(prov.client),
	}, diagnostics
}

type computeVolumeDataSource struct {
	volumeService compute.VolumeService
}

func (c computeVolumeDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeVolumeDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.volumeService.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list volumes: %s", err))
		return
	}

	volume, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find volume: %s", err))
		return
	}

	var state computeVolumeDataSourceData
	state.FromEntity(volume)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
