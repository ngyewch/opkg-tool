package main

import (
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"runtime/debug"
)

var (
	unameFlag = &cli.StringFlag{
		Name:  "uname",
		Usage: "override user name",
	}
	gnameFlag = &cli.StringFlag{
		Name:  "gname",
		Usage: "override group name",
	}
	uidFlag = &cli.IntFlag{
		Name:  "uid",
		Usage: "override user ID",
	}
	gidFlag = &cli.IntFlag{
		Name:  "gid",
		Usage: "override group ID",
	}
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
				Flags: []cli.Flag{
					unameFlag,
					gnameFlag,
					uidFlag,
					gidFlag,
				},
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

	buildInfo, _ := debug.ReadBuildInfo()
	if buildInfo != nil {
		app.Version = buildInfo.Main.Version
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
