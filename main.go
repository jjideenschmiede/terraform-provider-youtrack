// Copyright 2026 J&J Ideenschmiede GmbH. All rights reserved.

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/jjideenschmiede/terraform-provider-youtrack/internal/provider"
)

var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Start the provider with support for debuggers like delve.")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/jjideenschmiede/youtrack",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
