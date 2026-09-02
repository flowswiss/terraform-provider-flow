package flow

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// importStatePassthroughInt64ID is a drop-in replacement for
// tfsdk.ResourceImportStatePassthroughID for resources whose identifying
// attribute is of Int64 type. The upstream helper writes the import ID into
// the state as a string, which fails schema validation for numeric
// attributes with "Int64 Type Validation Error: Expected Number value,
// received tftypes.String".
func importStatePassthroughInt64ID(ctx context.Context, attrPath path.Path, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	id, err := strconv.ParseInt(request.ID, 10, 64)
	if err != nil {
		response.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("expected a numeric id, got %q: %s", request.ID, err),
		)
		return
	}

	response.Diagnostics.Append(response.State.SetAttribute(ctx, attrPath, types.Int64{Value: id})...)
}

// importStateCompositeInt64IDs is the same as importStatePassthroughInt64ID
// for resources that are only addressable through a parent: the import id carries every identifying
// attribute, colon separated, in the order of attrPaths (e.g. `server_id:id` → "42:7").
func importStateCompositeInt64IDs(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse, attrPaths ...path.Path) {
	parts := strings.Split(request.ID, ":")
	if len(parts) != len(attrPaths) {
		names := make([]string, len(attrPaths))
		for i, attrPath := range attrPaths {
			names[i] = attrPath.String()
		}
		response.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("expected %q, got %q", strings.Join(names, ":"), request.ID),
		)
		return
	}

	for i, part := range parts {
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			response.Diagnostics.AddError(
				"Invalid Import ID",
				fmt.Sprintf("expected a numeric %s, got %q: %s", attrPaths[i].String(), part, err),
			)
			return
		}
		response.Diagnostics.Append(response.State.SetAttribute(ctx, attrPaths[i], types.Int64{Value: id})...)
	}
}
