package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	"github.com/migalabs/goteth/cmd"
	"github.com/migalabs/goteth/pkg/utils"
)

var (
	log = logrus.WithField(
		"cli", "CliName",
	)
)

func main() {
	fmt.Println(utils.CliName, utils.Version)

	customFormatter := new(logrus.TextFormatter)
	customFormatter.FullTimestamp = true

	// Set the general log configurations for the entire tool
	// logrus.SetFormatter(utils.ParseLogFormatter("text"))
	logrus.SetFormatter(customFormatter)
	logrus.SetOutput(utils.ParseLogOutput("terminal"))
	logrus.SetLevel(utils.ParseLogLevel("info"))

	app := &cli.App{
		Name:      utils.CliName,
		Usage:     "Tinny client that requests and processes the Beacon State for the slot range defined.",
		UsageText: "eth2-state-analyzer [commands] [arguments...]",
		Authors: []*cli.Author{
			{
				Name:  "Cortze",
				Email: "cortze@protonmail.com",
			}, {
				Name:  "Tdahar",
				Email: "tarsuno@gmail.com",
			},
		},
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			cmd.BlocksCommand,
		},
	}

	// generate the crawler
	if err := app.RunContext(context.Background(), os.Args); err != nil {
		log.Errorf("error: %v\n", err)
		os.Exit(1)
	}
}
