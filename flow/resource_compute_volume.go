package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdk.ResourceType = (*computeVolumeResourceType)(nil)
var _ tfsdk.Resource = (*computeVolumeResource)(nil)
var _ tfsdk.ResourceWithImportState = (*computeVolumeResource)(nil)

type computeVolumeResourceData struct {
	ID           types.Int64  `tfsdk:"id"`
	SerialNumber types.String `tfsdk:"serial_number"`
	Name         types.String `tfsdk:"name"`
	Size         types.Int64  `tfsdk:"size"`
	Location     types.Int64  `tfsdk:"location"`

	Snapshot types.Int64 `tfsdk:"restore_from_snapshot"`
	Server   types.Int64 `tfsdk:"attach_to_server"`
}

func (c *computeVolumeResourceData) FromEntity(volume compute.Volume) {
	c.ID = types.Int64{Value: int64(volume.ID)}
	c.SerialNumber = types.String{Value: volume.SerialNumber}
	c.Name = types.String{Value: volume.Name}
	c.Size = types.Int64{Value: int64(volume.Size)}
	c.Location = types.Int64{Value: int64(volume.Location.ID)}
	c.Server = types.Int64{Value: int64(volume.AttachedTo.ID)}
}

type computeVolumeResourceType struct{}

func (c computeVolumeResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the volume",
				Computed:            true,
			},
			"serial_number": {
				Type:                types.StringType,
				MarkdownDescription: "unique serial number of the volume",
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the volume",
				Optional:            true,
			},
			"size": {
				Type:                types.Int64Type,
				MarkdownDescription: "size in GiB of the volume",
				Required:            true,
			},
			"location": {
				Type:                types.Int64Type,
				MarkdownDescription: "location of the volume",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"restore_from_snapshot": {
				Type:                types.Int64Type,
				MarkdownDescription: "restore the volume from the snapshot",
				Optional:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"attach_to_server": {
				Type:                types.Int64Type,
				MarkdownDescription: "server to attach the volume to",
				Optional:            true,
			},
		},
	}, nil
}

func (c computeVolumeResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return &computeVolumeResource{
		client: prov.client,
	}, diagnostics
}

type computeVolumeResource struct {
	client goclient.Client
}

func (c computeVolumeResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeVolumeResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.VolumeCreate{
		Name:       config.Name.Value,
		Size:       int(config.Size.Value),
		LocationID: int(config.Location.Value),
		SnapshotID: int(config.Snapshot.Value),
		InstanceID: int(config.Server.Value),
	}

	volume, err := compute.NewVolumeService(c.client).Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create volume: %s", err))
		return
	}

	var state computeVolumeResourceData
	state.FromEntity(volume)

	// copy the restored snapshot property from the config. in the api we don't know anymore if there was a snapshot
	// that has been restored.
	state.Snapshot = config.Snapshot

	tflog.Trace(ctx, "created volume", map[string]interface{}{
		"id": volume.ID,
	})

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeVolumeResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeVolumeResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	volume, err := compute.NewVolumeService(c.client).Get(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get volume: %s", err))
		return
	}

	state.FromEntity(volume)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeVolumeResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	var state computeVolumeResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeVolumeResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	volume, err := compute.NewVolumeService(c.client).Get(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get volume: %s", err))
		return
	}

	if config.Name.Value != volume.Name {
		update := compute.VolumeUpdate{
			Name: config.Name.Value,
		}

		volume, err = compute.NewVolumeService(c.client).Update(ctx, int(state.ID.Value), update)
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update volume: %s", err))
			return
		}
	}

	if int(config.Size.Value) < volume.Size {
		response.Diagnostics.AddWarning(
			"Volume resize not possible",
			"The requested volume size is smaller than current volume size. The volume will not be resized.",
		)
	}

	if int(config.Size.Value) > volume.Size {
		expand := compute.VolumeExpand{
			Size: int(config.Size.Value),
		}

		volume, err = compute.NewVolumeService(c.client).Expand(ctx, int(state.ID.Value), expand)
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to expand volume: %s", err))
			return
		}
	}

	state.FromEntity(volume)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeVolumeResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeVolumeResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	service := compute.NewVolumeService(c.client)

	if state.Server.Value != 0 {
		err := service.Detach(ctx, int(state.ID.Value), int(state.Server.Value))
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to detach volume: %s", err))
			return
		}
	}

	err := service.Delete(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete volume: %s", err))
		return
	}
}

func (c computeVolumeResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), request, response)
}
