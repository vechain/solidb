package cmd

import (
	"github.com/vechain/solidb/cmd/master"
	"github.com/vechain/solidb/cmd/node"
	cli "gopkg.in/urfave/cli.v1"
)

// Commands returns all commands
func Commands() []cli.Command {
	var ret []cli.Command
	for _, c := range node.Commands {
		c := c
		c.Category = "NODE"
		ret = append(ret, c)
	}

	for _, c := range master.Commands {
		c := c
		c.Category = "MASTER"
		ret = append(ret, c)
	}
	return ret
}
