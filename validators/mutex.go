package validators

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var _ tfsdk.ResourceConfigValidator = (*mutuallyExclusiveValidator)(nil)

type mutuallyExclusiveValidator struct {
	attributes []*tftypes.AttributePath
}

func MutuallyExclusive(attributes ...string) tfsdk.ResourceConfigValidator {
	attributePaths := make([]*tftypes.AttributePath, len(attributes))
	for i, attribute := range attributes {
		attributePaths[i] = tftypes.NewAttributePath().WithAttributeName(attribute)
	}

	return mutuallyExclusiveValidator{attributes: attributePaths}
}

func (m mutuallyExclusiveValidator) Description(ctx context.Context) string {
	attributeStrings := make([]string, len(m.attributes))
	for i, attribute := range m.attributes {
		attributeStrings[i] = attribute.String()
	}

	return fmt.Sprintf("attributes %s are mutually exclusive", strings.Join(attributeStrings, ", "))
}

func (m mutuallyExclusiveValidator) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m mutuallyExclusiveValidator) Validate(ctx context.Context, request tfsdk.ValidateResourceConfigRequest, response *tfsdk.ValidateResourceConfigResponse) {
	var previousAttributePath *tftypes.AttributePath = nil

	for _, attribute := range m.attributes {
		var value attr.Value

		diagnostics := request.Config.GetAttribute(ctx, attribute, &value)
		response.Diagnostics.Append(diagnostics...)
		if response.Diagnostics.HasError() {
			return
		}

		if value.IsUnknown() || value.IsNull() {
			continue
		}

		if previousAttributePath != nil {
			response.Diagnostics.AddAttributeError(
				attribute,
				"Mutually Exclusive Attribute Error",
				fmt.Sprintf("The attribute %s is mutually exclusive with %s. Please remove one of them.", attribute.String(), previousAttributePath.String()),
			)

			return
		}

		previousAttributePath = attribute
	}
}
