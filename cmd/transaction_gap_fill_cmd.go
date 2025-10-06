package cmd

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/analyzer"
	"github.com/migalabs/goteth/pkg/config"
	"github.com/migalabs/goteth/pkg/utils"
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

var transactionGapFillLog = logrus.WithField("module", "transactionGapFillCommand")

var TransactionGapFillCommand = &cli.Command{
	Name:   "transaction-gap-fill",
	Usage:  "reprocess blocks whose stored transactions count differs from block metrics",
	Action: LaunchTransactionGapFill,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "bn-endpoint",
			Usage:       "Beacon node endpoint (used to request blocks)",
			EnvVars:     []string{"ANALYZER_BN_ENDPOINT"},
			DefaultText: "http://localhost:5052",
		},
		&cli.StringFlag{
			Name:        "el-endpoint",
			Usage:       "Execution node endpoint (used to request receipts)",
			EnvVars:     []string{"ANALYZER_EL_ENDPOINT"},
			DefaultText: "http://localhost:8545",
		},
		&cli.StringFlag{
			Name:        "db-url",
			Usage:       "Clickhouse database url where metrics are stored",
			EnvVars:     []string{"ANALYZER_DB_URL"},
			DefaultText: "clickhouse://beaconchain:beaconchain@localhost:9000/beacon_states?x-multi-statement=true",
		},
		&cli.StringFlag{
			Name:        "log-level",
			Usage:       "Log level: debug, warn, info, error",
			EnvVars:     []string{"ANALYZER_LOG_LEVEL"},
			DefaultText: "info",
		},
		&cli.IntFlag{
			Name:        "max-request-retries",
			Usage:       "Number of retries when requesting data from the nodes",
			EnvVars:     []string{"ANALYZER_MAX_REQUEST_RETRIES"},
			DefaultText: "3",
		},
		&cli.StringFlag{
			Name:        "beacon-contract-address",
			Usage:       "Beacon contract address ('mainnet', 'holesky', 'sepolia' or 0x...)",
			EnvVars:     []string{"ANALYZER_BEACON_CONTRACT_ADDRESS"},
			DefaultText: "mainnet",
		},
		&cli.StringFlag{
			Name:    "metrics",
			Usage:   "Comma separated metrics list (transactions implied if missing)",
			Value:   "transactions",
			EnvVars: []string{"ANALYZER_METRICS"},
		},
		&cli.IntFlag{
			Name:  "limit",
			Usage: "Maximum number of mismatched slots to process (0 disables the limit)",
			Value: 0,
		},
	},
}

func LaunchTransactionGapFill(c *cli.Context) error {
	conf := config.NewTransactionGapConfig()
	conf.Apply(c)

	logrus.SetLevel(utils.ParseLogLevel(conf.LogLevel))

	filler, err := analyzer.NewTransactionGapFiller(c.Context, *conf)
	if err != nil {
		return err
	}
	defer filler.Close()

	gaps, err := filler.FindGaps()
	if err != nil {
		return err
	}
	if len(gaps) == 0 {
		transactionGapFillLog.Info("no transaction gaps detected")
		return nil
	}

	transactionGapFillLog.Infof("found %d transaction gap(s)", len(gaps))
	if conf.Limit > 0 && len(gaps) > conf.Limit {
		transactionGapFillLog.Infof("processing first %d gap(s) due to limit", conf.Limit)
		gaps = gaps[:conf.Limit]
	}

	for _, gap := range gaps {
		entry := transactionGapFillLog.WithField("slot", gap.Slot).
			WithField("expected", gap.Expected).
			WithField("actual", gap.Actual)

		entry.Info("reprocessing block to reconcile transactions")
		if err := filler.ReprocessSlot(phase0.Slot(gap.Slot)); err != nil {
			entry.WithError(err).Error("gap reprocessing failed")
			continue
		}
		entry.Info("gap reprocessing finished")
	}

	return nil
}
