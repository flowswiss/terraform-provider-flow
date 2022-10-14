package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/common"
	"github.com/flowswiss/goclient/kubernetes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.ResourceType            = (*kubernetesClusterResourceType)(nil)
	_ tfsdk.Resource                = (*kubernetesClusterResource)(nil)
	_ tfsdk.ResourceWithImportState = (*kubernetesClusterResource)(nil)
)

type kubernetesClusterResourceData struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	LocationID      types.Int64 `tfsdk:"location_id"`
	NetworkID       types.Int64 `tfsdk:"network_id"`
	SecurityGroupID types.Int64 `tfsdk:"security_group_id"`

	Public        types.Bool   `tfsdk:"public"`
	PublicAddress types.String `tfsdk:"public_address"`
	DNSName       types.String `tfsdk:"dns_name"`

	VersionID types.Int64 `tfsdk:"version_id"`

	NodeCount     types.Int64 `tfsdk:"node_count"`
	NodeProductID types.Int64 `tfsdk:"node_product_id"`
}

func (k *kubernetesClusterResourceData) FromEntity(cluster kubernetes.Cluster) {
	k.ID = types.Int64{Value: int64(cluster.ID)}
	k.Name = types.String{Value: cluster.Name}

	k.LocationID = types.Int64{Value: int64(cluster.Location.ID)}
	k.NetworkID = types.Int64{Value: int64(cluster.Network.ID)}
	k.SecurityGroupID = types.Int64{Value: int64(cluster.SecurityGroup.ID)}

	if cluster.PublicAddress == "" {
		k.Public = types.Bool{Value: false}
		k.PublicAddress = types.String{Null: true}
	} else {
		k.Public = types.Bool{Value: true}
		k.PublicAddress = types.String{Value: cluster.PublicAddress}
	}

	k.DNSName = types.String{Value: cluster.DNSName}

	k.VersionID = types.Int64{Value: int64(cluster.Version.ID)}

	k.NodeCount = types.Int64{Value: int64(cluster.NodeCount.Expected.Worker)}
	k.NodeProductID = types.Int64{Value: int64(cluster.ExpectedPreset.Worker.ID)}
}

type kubernetesClusterNameFilter struct {
	Name string
}

func (f kubernetesClusterNameFilter) AppliesTo(cluster kubernetes.Cluster) bool {
	return cluster.Name == f.Name
}

type kubernetesClusterResourceType struct{}

func (k kubernetesClusterResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the cluster",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the cluster",
				Required:            true,
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"network_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"security_group_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the security group",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"public": {
				Type:                types.BoolType,
				MarkdownDescription: "indicates if the cluster is public",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"public_address": {
				Type:                types.StringType,
				MarkdownDescription: "public address of the cluster",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"dns_name": {
				Type:                types.StringType,
				MarkdownDescription: "DNS name of the cluster",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"version_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the kubernetes version",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"node_count": {
				Type:                types.Int64Type,
				MarkdownDescription: "number of nodes in the cluster",
				Required:            true,
			},
			"node_product_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the node product",
				Required:            true,
			},
		},
	}, nil
}

func (k kubernetesClusterResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return kubernetesClusterResource{
		orderService:   common.NewOrderService(prov.client),
		clusterService: kubernetes.NewClusterService(prov.client),
	}, diagnostics
}

type kubernetesClusterResource struct {
	orderService   common.OrderService
	clusterService kubernetes.ClusterService
}

func (k kubernetesClusterResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config kubernetesClusterResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := kubernetes.ClusterCreate{
		Name:       config.Name.Value,
		LocationID: int(config.LocationID.Value),
		NetworkID:  int(config.NetworkID.Value),
		Worker: kubernetes.ClusterWorkerCreate{
			ProductID: int(config.NodeProductID.Value),
			Count:     int(config.NodeCount.Value),
		},
		AttachExternalIP: true,
	}

	if !config.Public.Null && !config.Public.Value {
		create.AttachExternalIP = false
	}

	ordering, err := k.clusterService.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create cluster: %s", err))
		return
	}

	order, err := k.orderService.WaitUntilProcessed(ctx, ordering)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster creation: %s", err))
		return
	}

	cluster, err := k.clusterService.Get(ctx, order.Product.ID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get cluster: %s", err))
		return
	}

	// set state of the resource
	var state kubernetesClusterResourceData
	state.FromEntity(cluster)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (k kubernetesClusterResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state kubernetesClusterResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	cluster, err := k.clusterService.Get(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get cluster: %s", err))
		return
	}

	state.FromEntity(cluster)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (k kubernetesClusterResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	var state kubernetesClusterResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config kubernetesClusterResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	if config.Name.Value != state.Name.Value {
		update := kubernetes.ClusterUpdate{
			Name: config.Name.Value,
		}

		_, err := k.clusterService.Update(ctx, int(state.ID.Value), update)
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update cluster: %s", err))
			return
		}
	}

	if config.VersionID.Value != state.VersionID.Value {
		update := kubernetes.ClusterConfiguration{
			VersionID: int(config.VersionID.Value),
			// TODO configuration options
		}

		_, err := k.clusterService.UpdateConfiguration(ctx, int(state.ID.Value), update)
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to change cluster configuration: %s", err))
			return
		}
	}

	if config.NodeCount.Value != state.NodeCount.Value || config.NodeProductID.Value != state.NodeProductID.Value {
		update := kubernetes.ClusterUpdateFlavor{
			Worker: kubernetes.ClusterWorkerUpdate{
				ProductID: int(config.NodeProductID.Value),
				Count:     int(config.NodeCount.Value),
			},
		}

		_, err := k.clusterService.UpdateFlavor(ctx, int(state.ID.Value), update)
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to change cluster flavor: %s", err))
			return
		}
	}

	cluster, err := k.clusterService.Get(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get cluster: %s", err))
		return
	}

	state.FromEntity(cluster)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (k kubernetesClusterResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state kubernetesClusterResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := k.clusterService.Delete(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete router: %s", err))
		return
	}
}

func (k kubernetesClusterResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
