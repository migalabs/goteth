package cmd

import (
	"sync"
	"sync/atomic"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/analyzer"
	"github.com/migalabs/goteth/pkg/config"
	"github.com/migalabs/goteth/pkg/utils"
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

var transactionGapFillLog = logrus.WithField("module", "transactionGapFillCommand")

type gapTask struct {
	slot     uint64
	expected *uint64
	actual   *uint64
	reason   string
}

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
		&cli.IntFlag{
			Name:  "workers",
			Usage: "Number of concurrent workers processing gaps",
			Value: config.DefaultTransactionGapWorkers,
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

	var processed int64
	limit := conf.Limit
	current := uint64(conf.StartSlot)
	batchSize := uint64(conf.BatchSize)
	workers := conf.Workers
	if workers <= 0 {
		workers = 1
	}

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

		tasks := make([]gapTask, 0, len(gaps))
		seenSlots := make(map[uint64]struct{}, len(gaps))
		for _, gap := range gaps {
			expectedCopy := gap.Expected
			actualCopy := gap.Actual
			tasks = append(tasks, gapTask{
				slot:     gap.Slot,
				expected: &expectedCopy,
				actual:   &actualCopy,
				reason:   "transaction_mismatch",
			})
			seenSlots[gap.Slot] = struct{}{}
		}

		missingSlots, err := filler.FindMissingBlockMetricsRange(current, end)
		if err != nil {
			return err
		}
		for _, slot := range missingSlots {
			if _, ok := seenSlots[slot]; ok {
				continue
			}
			tasks = append(tasks, gapTask{
				slot:   slot,
				reason: "missing_block_metrics",
			})
		}

		if len(tasks) == 0 {
			if end == lastSlotU {
				break
			}
			current = end + 1
			continue
		}

		limitReached := processGapTasksWithWorkers(tasks, filler, limit, &processed, workers)
		if limitReached {
			total := atomic.LoadInt64(&processed)
			transactionGapFillLog.Infof("processed %d gap(s); limit reached", total)
			return nil
		}

		if end == lastSlotU {
			break
		}
		current = end + 1
	}

	totalProcessed := atomic.LoadInt64(&processed)
	if totalProcessed == 0 {
		transactionGapFillLog.Info("no transaction gaps detected in scanned range")
	} else {
		transactionGapFillLog.Infof("processed %d gap(s)", totalProcessed)
	}

	return nil
}

func processGapTasksWithWorkers(
	tasks []gapTask,
	filler *analyzer.TransactionGapFiller,
	limit int,
	processed *int64,
	workers int,
) bool {
	if len(tasks) == 0 {
		return false
	}
	if workers <= 0 {
		workers = 1
	}
	jobs := make(chan gapTask)
	var wg sync.WaitGroup
	var enqueueLimitReached bool

	workerFn := func() {
		defer wg.Done()
		for task := range jobs {
			if limit > 0 && atomic.LoadInt64(processed) >= int64(limit) {
				return
			}

			entry := transactionGapFillLog.WithField("slot", task.slot).
				WithField("reason", task.reason)
			if task.expected != nil {
				entry = entry.WithField("expected", *task.expected)
			}
			if task.actual != nil {
				entry = entry.WithField("actual", *task.actual)
			}

			entry.Info("reprocessing block to reconcile transactions")
			if err := filler.ReprocessSlot(phase0.Slot(task.slot)); err != nil {
				entry.WithError(err).Error("gap reprocessing failed")
				continue
			}
			entry.Info("gap reprocessing finished")
			atomic.AddInt64(processed, 1)
		}
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go workerFn()
	}

	for _, task := range tasks {
		if limit > 0 && atomic.LoadInt64(processed) >= int64(limit) {
			enqueueLimitReached = true
			break
		}
		jobs <- task
	}
	close(jobs)
	wg.Wait()

	if limit > 0 && atomic.LoadInt64(processed) >= int64(limit) {
		return true
	}
	return enqueueLimitReached
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
