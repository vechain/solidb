package node

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vechain/solidb/broker"
	"github.com/vechain/solidb/kv"
	"github.com/vechain/solidb/node"
	"github.com/vechain/solidb/specmgr"
	"github.com/vechain/solidb/utils/fpath"
	cli "gopkg.in/urfave/cli.v1"
)

// DefaultHTTPPort default port of http service
const DefaultHTTPPort = 5670

var (
	Commands = []cli.Command{
		{
			Action: startNode,
			Name:   "node",
			Usage:  "start a node instance",
			Flags: []cli.Flag{
				bindFlag,
				dirFlag,
				devFlag,
			},
		},
	}
	bindFlag = cli.StringFlag{
		Name:  "bind",
		Usage: "IP:port binding of node",
		Value: fmt.Sprintf(":%d", DefaultHTTPPort),
	}
	dirFlag = cli.StringFlag{
		Name:  "dir",
		Usage: "dir of db",
	}
	devFlag = cli.BoolFlag{
		Name:   "dev",
		Usage:  "if set, node will use mem store",
		Hidden: true,
	}
)

func nodeDir(ctx *cli.Context) (string, error) {
	dir := ctx.String(dirFlag.Name)
	if dir == "" {
		home, err := fpath.HomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".solidb-node"), nil
	}
	if !filepath.IsAbs(dir) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(cwd, dir)
	} else {
		dir = filepath.Clean(dir)
	}
	return dir, nil
}

func startNode(ctx *cli.Context) error {
	log.Println(ctx.App.Name, ctx.App.Version)
	defer func() {
		log.Println("exited")
	}()

	listener, err := net.Listen("tcp", ctx.String(bindFlag.Name))
	if err != nil {
		return err
	}
	log.Println("HTTP server listening on", listener.Addr())

	var store kv.Store
	if ctx.IsSet(devFlag.Name) {
		store, err = kv.NewMemStore(kv.Options{CacheSize: 128})
		if err != nil {
			return err
		}
		log.Warnf("Runing in dev mode")
	} else {
		dataDir, err := nodeDir(ctx)
		if err != nil {
			return err
		}
		log.Println("Location:", dataDir)
		storePath := filepath.Join(dataDir, "store")
		store, err = kv.NewStore(storePath, kv.Options{
			CacheSize:              128,
			OpenFilesCacheCapacity: 32,
		})
		if err != nil {
			return err
		}
	}

	defer func() {
		store.Close()
		log.Println("store closed")
	}()
	specMgr := specmgr.New(store)
	n, err := node.New(store, specMgr)
	if err != nil {
		return err
	}
	log.Println("Node ID:", n.ID())
	log.Println("Cluster ID:", n.ClusterID())
	n.Start()
	defer n.Shutdown()

	brk := broker.New(store, specMgr)
	defer brk.Shutdown()

	mux := http.NewServeMux()
	mux.Handle(node.HTTPPathPrefix, node.NewHTTPHandler(n))
	mux.Handle(broker.HTTPPathPrefix, broker.NewHTTPHandler(brk))

	return serveHTTP(listener, mux)
}

func serveHTTP(listener net.Listener, handler http.Handler) error {
	server := &http.Server{
		Handler: handler,
	}
	errChan := make(chan error)
	go func() {
		if err := server.Serve(listener); err != nil {
			errChan <- err
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit,
		syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGHUP, syscall.SIGKILL,
		syscall.SIGUSR1, syscall.SIGUSR2)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	select {
	case sig := <-quit:
		log.Println("received", sig)
		return server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}
