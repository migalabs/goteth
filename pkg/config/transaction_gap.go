package config

import (
	"strings"

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

	if !containsMetric(c.Metrics, "transactions") {
		if strings.TrimSpace(c.Metrics) == "" {
			c.Metrics = "transactions"
		} else {
			c.Metrics = c.Metrics + ",transactions"
		}
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
