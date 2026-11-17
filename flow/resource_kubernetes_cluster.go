package flow

import (
	"context"
	"fmt"
	"net/http"

	"github.com/flowswiss/goclient/common"
	"github.com/flowswiss/goclient/compute"
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
					tfsdk.UseStateForUnknown(),
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

	var ordering common.Ordering
	err := retryCreate(ctx, "create cluster", func() (err error) {
		ordering, err = k.clusterService.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create cluster: %s", err))
		return
	}

	order, err := waitForOrder(ctx, k.orderService, ordering)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster creation: %s", err))
		return
	}

	cluster, err := waitForClusterReady(ctx, k.clusterService, order.Product.ID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster to be ready: %s", err))
		if cluster.ID == 0 {
			return
		}
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
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("cluster %d", state.ID.Value))
			return
		}
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

	var plan kubernetesClusterResourceData
	diagnostics = request.Plan.Get(ctx, &plan)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	if plan.Name.Value != state.Name.Value {
		// no unlock wait here — the name update is not guarded by the action , unlike configuration and flavor
		update := kubernetes.ClusterUpdate{
			Name: plan.Name.Value,
		}

		err := retry(ctx, "update cluster", func() (err error) {
			_, err = k.clusterService.Update(ctx, int(state.ID.Value), update)
			return err
		})
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update cluster: %s", err))
			return
		}
	}

	if !plan.VersionID.Unknown && plan.VersionID.Value != state.VersionID.Value {
		if _, err := k.waitForClusterUnlocked(ctx, int(state.ID.Value)); err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster to be unlocked: %s", err))
			return
		}

		current, err := k.clusterService.GetConfiguration(ctx, int(state.ID.Value))
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to read cluster configuration: %s", err))
			return
		}
		update := kubernetes.ClusterConfiguration{
			VersionID: int(plan.VersionID.Value),
			Variables: current.Variables,
		}

		err = retry(ctx, "update cluster configuration", func() (err error) {
			_, err = k.clusterService.UpdateConfiguration(ctx, int(state.ID.Value), update)
			return err
		})
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to change cluster configuration: %s", err))
			return
		}
	}

	if plan.NodeCount.Value != state.NodeCount.Value || plan.NodeProductID.Value != state.NodeProductID.Value {
		if _, err := k.waitForClusterUnlocked(ctx, int(state.ID.Value)); err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster to be unlocked: %s", err))
			return
		}

		update := kubernetes.ClusterUpdateFlavor{
			Worker: kubernetes.ClusterWorkerUpdate{
				ProductID: int(plan.NodeProductID.Value),
				Count:     int(plan.NodeCount.Value),
			},
		}

		err := retry(ctx, "update cluster flavor", func() (err error) {
			_, err = k.clusterService.UpdateFlavor(ctx, int(state.ID.Value), update)
			return err
		})
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to change cluster flavor: %s", err))
			return
		}
	}

	cluster, err := k.waitForClusterUnlocked(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster to be unlocked: %s", err))
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

	err := retryDelete(ctx, "delete cluster", func() error {
		return k.clusterService.Delete(ctx, int(state.ID.Value))
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete cluster: %s", err))
		return
	}

	if err := k.waitForClusterGone(ctx, int(state.ID.Value)); err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster deletion: %s", err))
		return
	}
}

// the create order succeeds while the cluster is still provisioning — until it
// is unlocked and healthy, updates are refused
func waitForClusterReady(ctx context.Context, service kubernetes.ClusterService, clusterID int) (cluster kubernetes.Cluster, err error) {
	err = waitFor(ctx, clusterWaitTimeout, defaultWaitInterval, fmt.Sprintf("cluster %d to be ready", clusterID), func(ctx context.Context) (bool, error) {
		got, err := service.Get(ctx, clusterID)
		if err != nil {
			return false, err
		}
		cluster = got
		return !cluster.Locked && cluster.Status.ID == compute.ClusterStatusHealthy, nil
	})

	return cluster, err
}

// configuration and flavor updates run as an async action that keeps the
// cluster locked after the call returns — any update in that window is refused
func (k kubernetesClusterResource) waitForClusterUnlocked(ctx context.Context, clusterID int) (cluster kubernetes.Cluster, err error) {
	err = waitFor(ctx, clusterWaitTimeout, defaultWaitInterval, fmt.Sprintf("cluster %d to be unlocked", clusterID), func(ctx context.Context) (bool, error) {
		got, err := k.clusterService.Get(ctx, clusterID)
		if err != nil {
			return false, err
		}
		cluster = got
		return !cluster.Locked, nil
	})

	return cluster, err
}

// cluster deletion is queued — the delete call returns while the cluster still
// exists, and deleting the network is refused until it is gone
func (k kubernetesClusterResource) waitForClusterGone(ctx context.Context, clusterID int) error {
	return waitFor(ctx, clusterWaitTimeout, defaultWaitInterval, fmt.Sprintf("cluster %d to be gone", clusterID), func(ctx context.Context) (bool, error) {
		_, err := k.clusterService.Get(ctx, clusterID)
		if statusCode(err) == http.StatusNotFound {
			return true, nil
		}
		return false, err
	})
}

func (k kubernetesClusterResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}
