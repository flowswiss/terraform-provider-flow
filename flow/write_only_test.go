package flow

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestComputeServerCreateSendsWriteOnlyValues(t *testing.T) {
	ctx := context.Background()

	var body []byte
	api := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ = io.ReadAll(request.Body)

		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = writer.Write([]byte(`{"error":{"message":{"en":"the fake api answers once"}}}`))
	}))
	defer api.Close()

	base, err := url.Parse(api.URL)
	if err != nil {
		t.Fatal(err)
	}
	server := &computeServerResource{
		client: newFlowClient(core.ClientOpts{BaseURL: base, HTTPClient: &http.Client{}, Token: "test"}),
	}

	var schemaResponse resource.SchemaResponse
	server.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	objectType := schemaResponse.Schema.Type().TerraformType(ctx)

	attributes := map[string]tftypes.Value{
		"name":        tftypes.NewValue(tftypes.String, "test-server"),
		"location_id": tftypes.NewValue(tftypes.Number, int64(1)),
		"image_id":    tftypes.NewValue(tftypes.Number, int64(2)),
		"product_id":  tftypes.NewValue(tftypes.Number, int64(3)),
		"network_id":  tftypes.NewValue(tftypes.Number, int64(4)),
		"key_pair_id": tftypes.NewValue(tftypes.Number, int64(5)),
	}
	plan := serverObject(t, objectType, attributes)

	attributes["password"] = tftypes.NewValue(tftypes.String, "sup3rs3cret")
	attributes["cloud_init"] = tftypes.NewValue(tftypes.String, "#cloud-config\n")
	config := serverObject(t, objectType, attributes)

	response := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResponse.Schema, Raw: tftypes.NewValue(objectType, nil)},
	}
	server.Create(ctx, resource.CreateRequest{
		Plan:   tfsdk.Plan{Schema: schemaResponse.Schema, Raw: plan},
		Config: tfsdk.Config{Schema: schemaResponse.Schema, Raw: config},
	}, response)

	if !response.Diagnostics.HasError() {
		t.Fatal("create succeeded against a fake api that only answers 401")
	}
	if len(body) == 0 {
		t.Fatal("no request reached the api")
	}

	var sent map[string]interface{}
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("request body is not json: %s", err)
	}

	for attribute, want := range map[string]string{
		"password":   "sup3rs3cret",
		"cloud_init": "#cloud-config\n",
	} {
		if got, ok := sent[attribute]; !ok || got != want {
			t.Errorf("%s on the wire is %v, want %q — Create read it from the plan, where it is null", attribute, got, want)
		}
	}
}

func serverObject(t *testing.T, objectType tftypes.Type, set map[string]tftypes.Value) tftypes.Value {
	t.Helper()

	object, ok := objectType.(tftypes.Object)
	if !ok {
		t.Fatalf("the schema is a %T, want an object", objectType)
	}

	values := make(map[string]tftypes.Value, len(object.AttributeTypes))
	for name, attributeType := range object.AttributeTypes {
		if value, ok := set[name]; ok {
			values[name] = value
			continue
		}
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	return tftypes.NewValue(objectType, values)
}
