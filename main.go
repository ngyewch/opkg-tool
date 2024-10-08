package main

import (
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

func main() {
	app := &cli.App{
		Name:  "opkg-tool",
		Usage: "opkg tool",
		Commands: []*cli.Command{
			{
				Name:      "create",
				Usage:     "create",
				ArgsUsage: "(ipk file) (input directory)",
				Action:    doCreate,
			},
			{
				Name:      "extract",
				Usage:     "extract",
				ArgsUsage: "(ipk file) (output directory)",
				Action:    doExtract,
			},
			{
				Name:      "view",
				Usage:     "view",
				ArgsUsage: "(ipk file)",
				Action:    doView,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
