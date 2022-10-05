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

var _ tfsdk.DataSourceType = (*computeImageDataSourceType)(nil)
var _ tfsdk.DataSource = (*computeImageDataSource)(nil)

type computeImageDataSourceData struct {
	ID              types.Int64  `tfsdk:"id"`
	OperatingSystem types.String `tfsdk:"operating_system"`
	Version         types.String `tfsdk:"version"`
	Key             types.String `tfsdk:"key"`
	Category        types.String `tfsdk:"category"`
	Type            types.String `tfsdk:"type"`
	Username        types.String `tfsdk:"username"`
	MinRootDiskSize types.Int64  `tfsdk:"min_root_disk_size"`
}

func (i *computeImageDataSourceData) FromEntity(image compute.Image) {
	i.ID = types.Int64{Value: int64(image.ID)}
	i.OperatingSystem = types.String{Value: image.OperatingSystem}
	i.Version = types.String{Value: image.Version}
	i.Key = types.String{Value: image.Key}
	i.Category = types.String{Value: image.Category}
	i.Type = types.String{Value: image.Type}
	i.Username = types.String{Value: image.Username}
	i.MinRootDiskSize = types.Int64{Value: int64(image.MinRootDiskSize)}
}

func (i computeImageDataSourceData) AppliesTo(image compute.Image) bool {
	if !i.ID.Null && image.ID != int(i.ID.Value) {
		return false
	}

	if !i.OperatingSystem.Null && image.OperatingSystem != i.OperatingSystem.Value {
		return false
	}

	if !i.Version.Null && image.Version != i.Version.Value {
		return false
	}

	if !i.Key.Null && image.Key != i.Key.Value {
		return false

	}

	if !i.Category.Null && image.Category != i.Category.Value {
		return false
	}

	if !i.Type.Null && image.Type != i.Type.Value {
		return false
	}

	return true
}

type computeImageDataSourceType struct{}

func (computeImageDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the image",
				Optional:            true,
				Computed:            true,
			},
			"operating_system": {
				Type:                types.StringType,
				MarkdownDescription: "operating system of the image",
				Optional:            true,
				Computed:            true,
			},
			"version": {
				Type:                types.StringType,
				MarkdownDescription: "version of the image",
				Optional:            true,
				Computed:            true,
			},
			"key": {
				Type:                types.StringType,
				MarkdownDescription: "unique key of the image",
				Optional:            true,
				Computed:            true,
			},
			"category": {
				Type:                types.StringType,
				MarkdownDescription: "category of the image (e.g. 'linux', 'windows')",
				Optional:            true,
				Computed:            true,
			},
			"type": {
				Type:                types.StringType,
				MarkdownDescription: "type of the image",
				Optional:            true,
				Computed:            true,
			},
			"username": {
				Type:                types.StringType,
				MarkdownDescription: "default username to connect to the server with",
				Computed:            true,
			},
			"min_root_disk_size": {
				Type:                types.Int64Type,
				MarkdownDescription: "minimum root disk size for servers using this image",
				Computed:            true,
			},
		},
	}, nil
}

func (computeImageDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeImageDataSource{
		imageService: compute.NewImageService(prov.client),
	}, diagnostics
}

type computeImageDataSource struct {
	imageService compute.ImageService
}

func (i computeImageDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeImageDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := i.imageService.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get images: %s", err))
		return
	}

	image, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find image: %s", err))
		return
	}

	var state computeImageDataSourceData
	state.FromEntity(image)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
