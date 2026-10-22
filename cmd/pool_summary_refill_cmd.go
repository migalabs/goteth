package cmd

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/db"
	"github.com/migalabs/goteth/pkg/utils"
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

var PoolSummaryRefillCommand = &cli.Command{
	Name:   "pool-summary-refill",
	Usage:  "Recomputes t_pool_summary for an epoch range from the current rewards and pool tags. Deletes the range first so pool renames do not leave ghost rows. Use after pool re-tagging or rewards backfills.",
	Action: LaunchPoolSummaryRefill,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "log-level",
			Usage:       "Log level: debug, warn, info, error",
			EnvVars:     []string{"ANALYZER_LOG_LEVEL"},
			DefaultText: "info",
		},
		&cli.StringFlag{
			Name:        "db-url",
			Usage:       "Database where the pool summaries are persisted",
			EnvVars:     []string{"ANALYZER_DB_URL"},
			DefaultText: "clickhouse://username:password@localhost:9000/goteth?x-multi-statement=true&max_memory_usage=10000000000",
		},
		&cli.Uint64Flag{
			Name:     "from-epoch",
			Usage:    "First epoch of the range to recompute (inclusive)",
			Required: true,
		},
		&cli.Uint64Flag{
			Name:     "to-epoch",
			Usage:    "Last epoch of the range to recompute (inclusive)",
			Required: true,
		},
	},
}

func LaunchPoolSummaryRefill(c *cli.Context) error {

	if c.IsSet("log-level") {
		logrus.SetLevel(utils.ParseLogLevel(c.String("log-level")))
	}

	dbUrl := c.String("db-url")
	if dbUrl == "" {
		return fmt.Errorf("db-url is required")
	}

	fromEpoch := phase0.Epoch(c.Uint64("from-epoch"))
	toEpoch := phase0.Epoch(c.Uint64("to-epoch"))
	if toEpoch < fromEpoch {
		return fmt.Errorf("to-epoch (%d) must be greater or equal than from-epoch (%d)", toEpoch, fromEpoch)
	}

	dbClient, err := db.New(c.Context, dbUrl)
	if err != nil {
		return err
	}
	err = dbClient.Connect()
	if err != nil {
		return err
	}
	defer dbClient.Finish()

	return dbClient.RefreshPoolSummaryRange(fromEpoch, toEpoch)
}
