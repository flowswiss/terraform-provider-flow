package flow

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var protoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"flow": providerserver.NewProtocol6WithError(New()),
}

func init() {
	version = "test"
}
