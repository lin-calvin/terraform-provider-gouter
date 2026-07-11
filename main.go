package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/anomalyco/gouter",
	}
	providerserver.Serve(context.Background(), func() provider.Provider {
		return &gouterProvider{}
	}, opts)
}
