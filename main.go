package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/vechain/solidb/cmd"
	"gopkg.in/urfave/cli.v1"
)

var (
	version   string
	gitCommit string
	release   = "dev"
)

func newApp() *cli.App {
	app := cli.NewApp()
	app.Version = fmt.Sprintf("%s-%s-commit%s", release, version, gitCommit)
	app.Name = "solidb"
	app.Usage = "A Distributed Content Addressable Database"
	app.Copyright = "2017 VeChain Foundation <https://vechain.com/>"
	app.Commands = cmd.Commands()
	return app
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "0102 15:04:05.000",
	})

	if err := newApp().Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
