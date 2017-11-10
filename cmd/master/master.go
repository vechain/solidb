package master

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/vechain/solidb/cmd/master/draft"
	"github.com/vechain/solidb/cmd/master/mod"
	ncmd "github.com/vechain/solidb/cmd/node"
	"github.com/vechain/solidb/crypto"
	"github.com/vechain/solidb/node"
	"github.com/vechain/solidb/utils/fpath"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	// Commands comannds exposed
	Commands = []cli.Command{
		{
			Action:    _new,
			Name:      "new",
			ArgsUsage: "path",
			Usage:     "create a new cluster",
			Flags: []cli.Flag{
				replicasFlag,
			},
		},
		{
			Action:    add,
			Name:      "add",
			ArgsUsage: "addr",
			Usage:     "add node to cluster",
			Flags: []cli.Flag{
				weightFlag,
			},
		},
		{
			Action:    remove,
			Name:      "remove",
			ArgsUsage: "index",
			Usage:     "remove node from cluster",
		},
		{
			Action:    alter,
			Name:      "alter",
			ArgsUsage: "index",
			Usage:     "alternate or replace a node",
			Flags: []cli.Flag{
				addrFlag,
				weightFlag,
			},
		},
		{
			Action: list,
			Name:   "list",
			Usage:  "list nodes in draft",
			Flags:  []cli.Flag{},
		},
		{
			Action: status,
			Name:   "status",
			Usage:  "query status of nodes in proposed spec",
			Flags:  []cli.Flag{},
		},
		{
			Action: propose,
			Name:   "propose",
			Usage:  "dispatch spec to nodes",
		},
		{
			Action: sync,
			Name:   "sync",
			Usage:  "notify nodes to sync",
		},
		{
			Action: approve,
			Name:   "approve",
			Usage:  "notify nodes that the spec has been approved",
		},
	}

	replicasFlag = cli.UintFlag{
		Name:  "replicas",
		Usage: "Replicas of solidb",
		Value: 2,
	}
	weightFlag = cli.UintFlag{
		Name:  "weight",
		Usage: "Weight of node",
		Value: 1,
	}
	addrFlag = cli.StringFlag{
		Name:  "addr",
		Usage: "Address of node",
	}
)

var errArgNum = errors.New("incorrect num of args")

const dirSuffix = ".solidb"

// create a new cluster
func _new(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		cli.ShowSubcommandHelp(ctx)
		return errArgNum
	}

	dir := ctx.Args().First()
	if !filepath.IsAbs(dir) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		dir = filepath.Join(cwd, dir)
	}
	dir = filepath.Clean(dir)
	base := filepath.Base(dir)
	if !strings.HasSuffix(base, dirSuffix) {
		base += dirSuffix
	}
	dir = filepath.Join(filepath.Dir(dir), base)

	if exists, err := fpath.PathExists(dir); err != nil {
		return err
	} else if exists {
		return errors.New("db exists")
	}

	replicas := ctx.Int(replicasFlag.Name)
	m, err := mod.New(dir, replicas)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	if err := m.Save(); err != nil {
		return err
	}
	fmt.Println("db created at", dir)
	return nil
}

func list(ctx *cli.Context) error {
	m, err := mod.Current()
	if err != nil {
		return err
	}

	fmt.Println("cluster ID:", m.Identity().ID())
	draft := m.Draft()
	fmt.Println("replicas:", draft.Replicas)

	for i, n := range draft.Nodes {
		fmt.Printf("[%d]\t%s\t%s\t%d\n", i, crypto.AbbrevID(n.ID), n.Addr, n.Weight)
	}

	return nil
}

func status(ctx *cli.Context) error {
	m, err := mod.Current()
	if err != nil {
		return err
	}
	proposed, err := m.LoadSpec(mod.StageProposed)
	if err != nil {
		return err
	}
	if proposed.V == nil {
		return errors.New("no proposed spec")
	}

	var nodeLocs []nodeLoc
	for _, entry := range proposed.V.SAT.Entries {
		nodeLocs = append(nodeLocs, nodeLoc{
			id:   entry.ID,
			addr: entry.Addr,
		})
	}

	statusChan := queryNodeStatus(nodeLocs)
	syncStatusChan := queryNodeSyncStatus(nodeLocs, proposed.V.Revision)

	i := 0
	for status := range statusChan {
		syncStatus := <-syncStatusChan
		fmt.Printf("%s\t%s\t%v\t%v\n", crypto.AbbrevID(nodeLocs[i].id), nodeLocs[i].addr, status, syncStatus)
		i++
	}
	return nil
}

func add(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		cli.ShowSubcommandHelp(ctx)
		return errArgNum
	}
	addr := ctx.Args().First()
	if strings.Index(addr, ":") < 0 {
		addr = addr + ":" + strconv.Itoa(ncmd.DefaultHTTPPort)
	}

	m, err := mod.Current()
	if err != nil {
		return err
	}

	weight := ctx.Int(weightFlag.Name)

	approved, err := m.LoadSpec(mod.StageApproved)
	if err != nil {
		return err
	}

	rpc := node.NewRPC().WithAddr(addr).WithIdentity(m.Identity(), "")
	nodeID, err := rpc.Invite(approved.V)
	if err != nil {
		return err
	}
	if err := m.AddNode(draft.Node{
		ID:     nodeID,
		Addr:   addr,
		Weight: weight,
	}); err != nil {
		return err
	}

	if err := m.Save(); err != nil {
		return err
	}
	fmt.Println("node added, ID", nodeID)
	return nil
}

func remove(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		cli.ShowSubcommandHelp(ctx)
		return errArgNum
	}
	m, err := mod.Current()
	if err != nil {
		return err
	}

	index, err := strconv.Atoi(ctx.Args().First())
	if err != nil {
		return err
	}
	if err := m.RemoveNode(index); err != nil {
		return err
	}
	return m.Save()
}

func alter(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		cli.ShowSubcommandHelp(ctx)
		return errArgNum
	}
	m, err := mod.Current()
	if err != nil {
		return err
	}

	index, err := strconv.Atoi(ctx.Args().First())
	if err != nil {
		return err
	}

	n, err := m.GetNode(index)
	if err != nil {
		return err
	}

	newNode := *n
	if ctx.IsSet(weightFlag.Name) {
		newNode.Weight = ctx.Int(weightFlag.Name)
	}

	if ctx.IsSet(addrFlag.Name) {
		newNode.Addr = ctx.String(addrFlag.Name)
	}

	if err := m.AlterNode(index, newNode); err != nil {
		return err
	}
	return m.Save()
}

func propose(ctx *cli.Context) error {
	m, err := mod.Current()
	if err != nil {
		return err
	}
	s, err := m.BuildSpec()
	if err != nil {
		return err
	}
	rpc := node.NewRPC()
	for _, e := range s.SAT.Entries {
		rpc := rpc.WithAddr(e.Addr).WithIdentity(m.Identity(), e.ID)
		if err := rpc.ProposeSpec(*s); err != nil {
			return err
		}
	}

	if err := m.SaveSpec(mod.StageProposed, *s); err != nil {
		return err
	}
	if s.Revision == 0 {
		return m.SaveSpec(mod.StageApproved, *s)
	}
	return nil
}

func sync(ctx *cli.Context) error {
	m, err := mod.Current()
	if err != nil {
		return err
	}
	proposed, err := m.LoadSpec(mod.StageProposed)
	if err != nil {
		return err
	}
	if proposed.V == nil {
		return errors.New("no proposed spec")
	}
	rpc := node.NewRPC()
	for _, entry := range proposed.V.SAT.Entries {
		rpc := rpc.WithAddr(entry.Addr).WithIdentity(m.Identity(), entry.ID)
		if err := rpc.SyncToSpec(proposed.V.Revision); err != nil {
			return err
		}
	}
	return nil
}

func approve(ctx *cli.Context) error {
	m, err := mod.Current()
	if err != nil {
		return err
	}
	proposed, err := m.LoadSpec(mod.StageProposed)
	if err != nil {
		return err
	}
	if proposed.V == nil {
		return errors.New("no proposed spec")
	}

	var nodeLocs []nodeLoc
	for _, entry := range proposed.V.SAT.Entries {
		nodeLocs = append(nodeLocs, nodeLoc{
			id:   entry.ID,
			addr: entry.Addr,
		})
	}

	statusChan := queryNodeStatus(nodeLocs)
	for status := range statusChan {
		if status.err != nil {
			return errors.Wrap(err, "query status")
		}
		if status.status.SpecRevisions.Synced != proposed.V.Revision {
			return errors.New("not synced")
		}
	}

	rpc := node.NewRPC()
	for _, entry := range proposed.V.SAT.Entries {
		rpc := rpc.WithAddr(entry.Addr).WithIdentity(m.Identity(), entry.ID)
		if err := rpc.ApproveSpec(proposed.V.Revision); err != nil {
			return err
		}
	}
	return m.SaveSpec(mod.StageApproved, *proposed.V)
}
