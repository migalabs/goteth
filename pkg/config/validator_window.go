package config

import (
	cli "github.com/urfave/cli/v2"
)

type ValidatorWindowConfig struct {
	LogLevel          string `json:"log-level"`
	DBUrl             string `json:"db-url"`
	NumEpochs         int    `json:"num-epochs"`
	BnEndpoint        string `json:"bn-endpoint"`
	BnApiKey          string `json:"bn-api-key"`
	MaxRequestRetries int    `json:"max-request-retries"`
}

// TODO: read from config-file
func NewValidatorWindowConfig() *ValidatorWindowConfig {
	// Return Default values for the ethereum configuration
	return &ValidatorWindowConfig{
		LogLevel:          DefaultLogLevel,
		DBUrl:             DefaultDBUrl,
		NumEpochs:         DefaultValidatorWindowEpochs,
		BnEndpoint:        DefaultBnEndpoint,
		MaxRequestRetries: DefaultMaxRequestRetries,
		BnApiKey:          DefaultBnApiKey,
	}
}

func (c *ValidatorWindowConfig) Apply(ctx *cli.Context) {
	// apply to the existing Default configuration the set flags
	// log level
	if ctx.IsSet("log-level") {
		c.LogLevel = ctx.String("log-level")
	}
	// db url
	if ctx.IsSet("db-url") {
		c.DBUrl = ctx.String("db-url")
	}
	// validator window epochs
	if ctx.IsSet("num-epochs") {
		c.NumEpochs = ctx.Int("num-epochs")
	}
	// cl url
	if ctx.IsSet("bn-endpoint") {
		c.BnEndpoint = ctx.String("bn-endpoint")
	}
	// bn api key
	if ctx.IsSet("bn-api-key") {
		c.BnApiKey = ctx.String("bn-api-key")
	}

	// max request retries
	if ctx.IsSet("max-request-retries") {
		c.MaxRequestRetries = ctx.Int("max-request-retries")
	}

}
