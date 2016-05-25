package main

import (
	"flag"
	"fmt"

	"github.com/rtnpro/henge/pkg/loaders/compose"
	"github.com/rtnpro/henge/pkg/transformers"
)

func main() {
	provider := flag.String("provider", "openshift", "Target provider")

	flag.Parse()

	project, err = compose.Load(flag.Args()[0:]...)

	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	transformers.Transform(provider, project)
}
