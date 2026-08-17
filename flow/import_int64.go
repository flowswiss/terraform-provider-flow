package flow

import (
	"context"
	"fmt"
	"strconv"

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
