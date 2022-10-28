# Eth CL State Analyzer

The CL State Analyzer is a go-written client that indexes all validator-related duties and parameters from Ethereum's beaconchain by fetching the CL States from a node (preferable a locally running archival node).

The client indexes all the validator/epoch related metrics into a set of postgreSQL tables. Which later on can be used to monitor the performance of validators in the beaconchain.

This tool has been used to power the [pandametrics.xyz](https://pandametrics.xyz/) public dashboard.

### Prerequisites
To use the tool, the following requirements need to be installed in the machine:
- [go](https://go.dev/doc/install) preferably on its 1.17 version or above. Go also needs to be executable from the terminal.
- PostgreSQL DB
- Access to a Ethereum CL beacon node (preferably an archive node to index the slots faster)

### Installation
The repository provides a Makefile that will take care of all your problems.

To compile locally the client, just type the following command at the root of the directory:
```
make build
```

Or if you prefer to install the client locally type:
```
make install
```

### Running the tool
To execute the tool, you can simply modify the `.env` file with your own configuration. The `.env` file first exports all the variables as system environment variables, and then uses them as arguments when calling the tool.

*Running the tool (configurable in the `.env` file)*:
```
make run
```

*Available Commands*:
```
COMMANDS:
   rewards  analyze the Beacon State of a given slot range
   help, h  Shows a list of commands or help for one command
```

*Available Options (configurable in the `.env` file)*
```
OPTIONS:
   --bn-endpoint value        beacon node endpoint (to request the BeaconStates)
   --init-slot value          init slot from where to start (default: 0)
   --final-slot value         init slot from where to finish (default: 0)
   --validator-indexes value  json file including the list of validator indexes (leave the json `[]` to index all the existing validators)
   --log-level value          log level: debug, warn, info, error
   --db-url value             example: postgresql://beaconchain:beaconchain@localhost:5432/beacon_states
   --workers-num value        example: 50 (default: 0)
   --db-workers-num value     example: 50 (default: 0)
   --help, -h                 show help (default: false)
```


# Maintainers
@cortze , @tadahar

# Contributing
The project is open for everyone to contribute! 