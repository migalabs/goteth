package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	"github.com/cortze/eth2-state-analyzer/cmd"
	"github.com/cortze/eth2-state-analyzer/pkg/utils"
)

var (
	Version = "v0.0.1"
	CliName = "Eth2 State Analyzer"
	log     = logrus.WithField(
		"cli", "CliName",
	)
)

func main() {
	fmt.Println(CliName, Version)

	//ctx, cancel := context.WithCancel(context.Background())

	// Set the general log configurations for the entire tool
	logrus.SetFormatter(utils.ParseLogFormatter("text"))
	logrus.SetOutput(utils.ParseLogOutput("terminal"))
	logrus.SetLevel(utils.ParseLogLevel("info"))

	app := &cli.App{
		Name:      CliName,
		Usage:     "Tinny client that requests and processes the Beacon State for the slot range defined.",
		UsageText: "state-analyzer [commands] [arguments...]",
		Authors: []*cli.Author{
			{
				Name:  "Cortze",
				Email: "cortze@protonmail.com",
			},
		},
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			cmd.RewardsCommand,
		},
	}

	// generate the crawler
	if err := app.RunContext(context.Background(), os.Args); err != nil {
		log.Errorf("error: %v\n", err)
		os.Exit(1)
	}
}
