package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/migalabs/goteth/pkg/config"
	"github.com/migalabs/goteth/pkg/utils"
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	"github.com/migalabs/goteth/pkg/analyzer"
)

var BlocksCommand = &cli.Command{
	Name:   "blocks",
	Usage:  "analyze the Beacon Block of a given slot range",
	Action: LaunchBlockMetrics,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "bn-endpoint",
			Usage:       "Beacon node endpoint (to request the Beacon States and Blocks)",
			EnvVars:     []string{"ANALYZER_BN_ENDPOINT"},
			DefaultText: "http://localhost:5052",
		},
		&cli.StringFlag{
			Name:        "el-endpoint",
			Usage:       "Execution node endpoint (to request more specific data on Blocks)",
			EnvVars:     []string{"ANALYZER_EL_ENDPOINT"},
			DefaultText: "http://localhost:8545",
		},
		&cli.IntFlag{
			Name:        "init-slot",
			Usage:       "Slot from where to start the backfill",
			EnvVars:     []string{"ANALYZER_INIT_SLOT"},
			DefaultText: "0",
		},
		&cli.IntFlag{
			Name:        "final-slot",
			Usage:       "Slot from where to finish the backfill",
			EnvVars:     []string{"ANALYZER_FINAL_SLOT"},
			DefaultText: "0",
		},
		&cli.IntFlag{
			Name:        "rewards-aggregation-epochs",
			Usage:       "Number of epochs to aggregate rewards",
			EnvVars:     []string{"ANALYZER_REWARDS_AGGREGATION_EPOCHS"},
			DefaultText: "1 (no aggregation)",
		},
		&cli.StringFlag{
			Name:        "log-level",
			Usage:       "Log level: debug, warn, info, error",
			EnvVars:     []string{"ANALYZER_LOG_LEVEL"},
			DefaultText: "info",
		},
		&cli.StringFlag{
			Name:        "db-url",
			Usage:       "Clickhouse database url where to persist the metrics",
			EnvVars:     []string{"ANALYZER_DB_URL"},
			DefaultText: "clickhouse://beaconchain:beaconchain@localhost:9000/beacon_states?x-multi-statement=true",
		},
		&cli.IntFlag{
			Name:        "workers-num",
			Usage:       "Number of workers to process validators",
			EnvVars:     []string{"ANALYZER_WORKER_NUM"},
			DefaultText: "4",
		},
		&cli.IntFlag{
			Name:        "db-workers-num",
			Usage:       "Number of workers to process database operations",
			EnvVars:     []string{"ANALYZER_DB_WORKER_NUM"},
			DefaultText: "4",
		},
		&cli.StringFlag{
			Name:        "download-mode",
			Usage:       "Either backfill specified slots or follow the chain head example: hybrid,historical,finalized",
			EnvVars:     []string{"ANALYZER_DOWNLOAD_MODE"},
			DefaultText: "finalized",
		},
		&cli.StringFlag{
			Name:        "metrics",
			Usage:       "Metrics to be persisted to the database: epoch,block,rewards,transactions,api_rewards",
			EnvVars:     []string{"ANALYZER_METRICS"},
			DefaultText: "epoch,block",
		},
		&cli.IntFlag{
			Name:        "prometheus-port",
			Usage:       "Port on which to expose prometheus metrics",
			EnvVars:     []string{"ANALYZER_PROMETHEUS_PORT"},
			DefaultText: "9080",
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

	logrus.SetLevel(utils.ParseLogLevel(conf.LogLevel))

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
