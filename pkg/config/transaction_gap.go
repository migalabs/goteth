package config

import (
	"strings"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	cli "github.com/urfave/cli/v2"
)

type TransactionGapConfig struct {
	LogLevel              string `json:"log-level"`
	BnEndpoint            string `json:"bn-endpoint"`
	ElEndpoint            string `json:"el-endpoint"`
	DBUrl                 string `json:"db-url"`
	MaxRequestRetries     int    `json:"max-request-retries"`
	BeaconContractAddress string `json:"beacon-contract-address"`
	Metrics               string `json:"metrics"`
	Limit                 int    `json:"limit"`
	StartSlot             phase0.Slot
	BatchSize             int
	Workers               int
}

func NewTransactionGapConfig() *TransactionGapConfig {
	return &TransactionGapConfig{
		LogLevel:              DefaultLogLevel,
		BnEndpoint:            DefaultBnEndpoint,
		ElEndpoint:            DefaultElEndpoint,
		DBUrl:                 DefaultDBUrl,
		MaxRequestRetries:     DefaultMaxRequestRetries,
		BeaconContractAddress: DefaultBeaconContractAddress,
		Metrics:               "transactions",
		Limit:                 0,
		StartSlot:             0,
		BatchSize:             DefaultTransactionGapBatchSize,
		Workers:               DefaultTransactionGapWorkers,
	}
}

func (c *TransactionGapConfig) Apply(ctx *cli.Context) {
	if ctx.IsSet("log-level") {
		c.LogLevel = ctx.String("log-level")
	}
	if ctx.IsSet("bn-endpoint") {
		c.BnEndpoint = ctx.String("bn-endpoint")
	}
	if ctx.IsSet("el-endpoint") {
		c.ElEndpoint = ctx.String("el-endpoint")
	}
	if ctx.IsSet("db-url") {
		c.DBUrl = ctx.String("db-url")
	}
	if ctx.IsSet("max-request-retries") {
		c.MaxRequestRetries = ctx.Int("max-request-retries")
	}
	if ctx.IsSet("beacon-contract-address") {
		c.BeaconContractAddress = ctx.String("beacon-contract-address")
	}
	if ctx.IsSet("metrics") {
		c.Metrics = ctx.String("metrics")
	}
	if ctx.IsSet("limit") {
		c.Limit = ctx.Int("limit")
	}
	if ctx.IsSet("start-slot") {
		c.StartSlot = phase0.Slot(ctx.Uint64("start-slot"))
	}
	if ctx.IsSet("batch-size") {
		c.BatchSize = ctx.Int("batch-size")
	}
	if ctx.IsSet("workers") {
		c.Workers = ctx.Int("workers")
	}

	if !containsMetric(c.Metrics, "transactions") {
		if strings.TrimSpace(c.Metrics) == "" {
			c.Metrics = "transactions"
		} else {
			c.Metrics = c.Metrics + ",transactions"
		}
	}
	if c.BatchSize <= 0 {
		c.BatchSize = DefaultTransactionGapBatchSize
	}
	if c.Workers <= 0 {
		c.Workers = 1
	}
}

func containsMetric(metrics string, target string) bool {
	for _, item := range strings.Split(metrics, ",") {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}
