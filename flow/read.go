package flow

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// an object deleted outside terraform (e.g. via API) is dropped from the state on refresh,
// so the next plan recreates it instead of failing

func isNotFound(err error) bool {
	return statusCode(err) == http.StatusNotFound
}

func removeGone(ctx context.Context, response *resource.ReadResponse, what string) {
	tflog.Debug(ctx, "object gone, removing from state", map[string]interface{}{"object": what})
	response.State.RemoveResource(ctx)
}
