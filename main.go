package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/me/dkg-node/config"
	"github.com/me/dkg-node/services"
	"github.com/sirupsen/logrus"
)

var path string

func init() {
	flag.StringVar(&path, "config-path", "./config/config.json", "config file")

	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

func main() {
	flag.Parse()
	ctx := context.Background()

	// Load config
	globalConfig, err := config.LoadConfig(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v", err)
		os.Exit(1)
	}
	nodeList, err := config.LoadNodeList()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v", err)
		os.Exit(1)
	}
	config.GlobalConfig = globalConfig
	config.NodeList = nodeList

	// Initial service
	ethereumService := services.NewEthereumService(ctx)
	abciService := services.NewABCIService(ctx)
	p2pService := services.NewP2PService(ctx)
	keyGenService := services.NewKeyGenService(ctx)
	verifierService := services.NewVerifierService(ctx)
	tendermintService := services.NewTendermintService(ctx)

	compositeService := services.NewCompositeService(ethereumService, abciService, p2pService, keyGenService, verifierService, tendermintService)
	services.GlobalCompositeService = compositeService
	// Start all services
	err = services.GlobalCompositeService.OnStart()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start composite service:%v", err)
		os.Exit(1)
	}

	// Inject services for service after start
	keyGenService.InjectServices(p2pService, abciService.ABCIApp)

	// Initialize all necessary channels
	nodeListMonitorTicker := time.NewTicker(5 * time.Second)
	establishConnection := make(chan bool)
	services.TestPublicKey()

	go services.SetUpJRPCHandler()
	go services.NodeListMonitor(nodeListMonitorTicker.C, p2pService, establishConnection)
	<-establishConnection
	services.KeyGenStart(keyGenService)
	// Stop NodeList monitor ticker
	nodeListMonitorTicker.Stop()

	// Exit the blocking chan
	close(establishConnection)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	err = compositeService.OnStop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not stop composite service:%v", err)
	}
	os.Exit(0)
}
