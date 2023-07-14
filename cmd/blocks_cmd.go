package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cortze/eth-cl-state-analyzer/pkg/config"
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	"github.com/cortze/eth-cl-state-analyzer/pkg/analyzer"
)

var BlocksCommand = &cli.Command{
	Name:   "blocks",
	Usage:  "analyze the Beacon Block of a given slot range",
	Action: LaunchBlockMetrics,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "bn-endpoint",
			Usage: "beacon node endpoint (to request the Beacon States and Blocks)",
		},
		&cli.StringFlag{
			Name:  "el-endpoint",
			Usage: "execution node endpoint (to request more specific data on Blocks)",
		},
		&cli.IntFlag{
			Name:  "init-slot",
			Usage: "init slot from where to start",
		},
		&cli.IntFlag{
			Name:  "final-slot",
			Usage: "init slot from where to finish",
		},
		&cli.StringFlag{
			Name:  "log-level",
			Usage: "log level: debug, warn, info, error",
		},
		&cli.StringFlag{
			Name:  "db-url",
			Usage: "example: postgresql://beaconchain:beaconchain@localhost:5432/beacon_states",
		},
		&cli.IntFlag{
			Name:  "workers-num",
			Usage: "example: 50",
		},
		&cli.IntFlag{
			Name:  "db-workers-num",
			Usage: "example: 50",
		},
		&cli.StringFlag{
			Name:  "download-mode",
			Usage: "example: hybrid,historical,finalized. Default: hybrid",
		},
		&cli.StringFlag{
			Name:        "metrics",
			Usage:       "example: epoch,validator, epoch. Empty for all",
			DefaultText: "epoch",
		},
		&cli.IntFlag{
			Name:  "prometheus-port",
			Usage: "example: 9080",
		}},
}

var logCmdChain = logrus.WithField(
	"module", "chainCommand",
)

var QueryTimeout = 90 * time.Second

// CrawlAction is the function that is called when running `eth2`.
func LaunchBlockMetrics(c *cli.Context) error {

	conf := config.NewAnalyzerConfig()
	conf.Apply(c)

	// generate the block analyzer
	blockAnalyzer, err := analyzer.NewChainAnalyzer(c.Context, *conf)
	if err != nil {
		return err
	}

	procDoneC := make(chan struct{})
	sigtermC := make(chan os.Signal, 1)

	signal.Notify(sigtermC, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGTERM)

	go func() {
		blockAnalyzer.Run()
		procDoneC <- struct{}{}
	}()

	select {
	case <-sigtermC:
		logCmdChain.Info("Sudden shutdown detected, controlled shutdown of the cli triggered")
		blockAnalyzer.Close()

	case <-procDoneC:
		logCmdChain.Info("Process successfully finish!")
	}
	close(sigtermC)
	close(procDoneC)

	return nil
}
