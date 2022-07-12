package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/flowswiss/terraform-provider-flow/flow"
)

var version = "dev"

func main() {
	debug := false

	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address:         "registry.terraform.io/flowswiss/flow",
		Debug:           debug,
		ProtocolVersion: 6,
	}

	err := providerserver.Serve(context.Background(), flow.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
