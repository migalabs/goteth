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
		&cli.Uint64Flag{
			Name:  "start-slot",
			Usage: "Slot from which to start scanning for gaps",
			Value: 0,
		},
		&cli.IntFlag{
			Name:  "batch-size",
			Usage: "Number of slots to scan per batch when searching for gaps",
			Value: config.DefaultTransactionGapBatchSize,
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

	lastSlot, err := filler.LastSlot()
	if err != nil {
		return err
	}

	if conf.StartSlot > lastSlot {
		transactionGapFillLog.Infof("start slot %d is beyond last stored slot %d; nothing to process", conf.StartSlot, lastSlot)
		return nil
	}

	processed := 0
	limit := conf.Limit
	current := uint64(conf.StartSlot)
	batchSize := uint64(conf.BatchSize)
	lastSlotU := uint64(lastSlot)

	for current <= lastSlotU {
		end := min(current+batchSize-1, lastSlotU)

		transactionGapFillLog.WithField("start_slot", current).
			WithField("end_slot", end).
			Info("scanning for transaction gaps")

		gaps, err := filler.FindGapsRange(current, end)
		if err != nil {
			return err
		}

		if len(gaps) == 0 {
			if end == lastSlotU {
				break
			}
			current = end + 1
			continue
		}

		for _, gap := range gaps {
			if limit > 0 && processed >= limit {
				transactionGapFillLog.Infof("processed %d gap(s); limit reached", processed)
				return nil
			}
			entry := transactionGapFillLog.WithField("slot", gap.Slot).
				WithField("expected", gap.Expected).
				WithField("actual", gap.Actual)

			entry.Info("reprocessing block to reconcile transactions")
			if err := filler.ReprocessSlot(phase0.Slot(gap.Slot)); err != nil {
				entry.WithError(err).Error("gap reprocessing failed")
				continue
			}
			entry.Info("gap reprocessing finished")
			processed++
		}

		if limit > 0 && processed >= limit {
			transactionGapFillLog.Infof("processed %d gap(s); limit reached", processed)
			return nil
		}

		if end == lastSlotU {
			break
		}
		current = end + 1
	}

	if processed == 0 {
		transactionGapFillLog.Info("no transaction gaps detected in scanned range")
	} else {
		transactionGapFillLog.Infof("processed %d gap(s)", processed)
	}

	return nil
}
