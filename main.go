package main

import (
	"context"
	"flag"
	"log"

	"github.com/OrcaCD/terraform-provider-orcacd/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "dev"

func main() {
	debug := flag.Bool("debug", false, "run with debugger support")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/orcacd/orcacd",
		Debug:   *debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
