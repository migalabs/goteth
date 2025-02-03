# GotEth

GotEth is a go-written client that indexes all validator-related duties and parameters from Ethereum's beaconchain by fetching the CL States from a node (preferable a locally running archival node).

The client indexes all the validator/epoch related metrics into a set of clickhouse tables which later on can be used to monitor the performance of validators in the beaconchain.

This tool has been used to power the

- [pandametrics.xyz](https://pandametrics.xyz/) public dashboard
- [ethseer.io](https://ethseer.io) block explorer
- [monitoreth.io](https://monitoreth.io/nodes#validators-entities) public dashboard

## Prerequisites

To use the tool, the following requirements need to be installed in the machine:

- [go](https://go.dev/doc/install) preferably on its 1.21 version or above. Go also needs to be executable from the terminal.
- Clickhouse DB
- Access to an Ethereum CL beacon node (preferably an archive node to index the slots faster)
- Access to an Ethereum execution node (optional)
- Access to a Clickhouse server database (use native port, usually 9000)

## Installation

The repository provides a Makefile that will take care of all your problems.

To compile locally the client, just type the following command at the root of the directory:

```
make build
```

Or if you prefer to install the client locally type:

```
make install
```

## Metrics: database tables

- block: downloads withdrawals, blocks and block rewards
- epoch: download epoch metrics, proposer duties, validator last status,
- rewards: persists validator rewards metrics to database (activates epoch metrics)
- api_rewards (EXPERIMENTAL): block rewards (consensus layer) are hard to calculate, but they can be downloaded from the Beacon API. However, keep in mind this takes a few seconds per block when not at the head. Without this, reward cannot be compared to max_reward when a validator is a proposer (32/900K validators in an epoch). It depends on the Lighthouse API and we have registered some cases where the block reward was not returned.
- transactions: requests transaction receipts from the execution layer (activates block metrics)

Go to [docs/tables.md](https://github.com/migalabs/goteth/blob/master/docs/tables.md) for more information on the tables indexed by Goteth.

## Download mode

- Historical: this mode loops over slots between `initSlot` and `finalSlot`, which are configurable. Once all slots have been analyzed, the tool finishes the execution.
- Finalized: `initSlot` and `finalSlot` are ignored. The tool starts the historical mode from the database last slot to the current head (beacon node) and then follows the chain head. To do this, the tool subscribes to `head` events. See [here](https://ethereum.github.io/beacon-APIs/#/Events/eventstream) for more information.

## Running the tool

To execute the tool, you can simply modify the `.env` file with your own configuration.

_Running the tool (configurable in the `.env` file)_:

```
docker-compose up goteth
```

_Available Commands_:

```
COMMANDS:
   blocks   analyze the Beacon Block of a given slot range
   val-window Removes old rows from the validator rewards table according to given parameters
   help, h  Shows a list of commands or help for one command
```

_Available Options (configurable in the `.env` file)_

```

Blocks
OPTIONS:
   --bn-endpoint value     beacon node endpoint (to request the Beacon Blocks)
   --el-endpoint value 	   execution node endpoint (to request the Transaction Receipts, optional)
   --init-slot value       init slot from where to start (default: 0)
   --final-slot value      init slot from where to finish (default: 0)
   --rewards-aggregation-epochs value  Number of epochs to aggregate rewards (default: 1 (no aggregation))
   --log-level value       log level: debug, warn, info, error
   --db-url value          example: clickhouse://beaconchain:beaconchain@localhost:9000/beacon_states?x-multi-statement=true
   --workers-num value     example: 3 (default: 4)
   --db-workers-num value  example: 3 (default: 4)
   --download-mode value   example: hybrid,historical,finalized. Default: hybrid
   --metrics value         example: epoch,block,rewards,transactions,api_rewards. Empty for all (default: epoch,block)
   --prometheus-port value Port on which to expose prometheus metrics (default: 9081)
   --max-request-retries value         Number of retries to make when a request fails. For head mode it shouldn't be higher than 3-4, for historical its recommended to be higher (default: 3)
   --help, -h              show help (default: false)
```

### Validator window (experimental)

Validator rewards represent 95% of the disk usage of the database. When activated, the database grows very big, sometimes becoming too much data.
We have developed a subcommand of the tool which maintains the last n epochs of rewards data in the database, prunning from the defined threshold backwards. So, one can configure the tool to maintain the last 100 epochs of data in the database, while prunning the rest.
The pruning only affects the `t_validator_rewards_summary` table.

Simply configure `GOTETH_VAL_WINDOW_NUM_EPOCHS` variable and run

```
docker-compose up val-window
```

# Notes

Keep in mind `api_rewards` data also downloads block rewards from the Beacon API. This is very slow on historical blocks (3 seconds per block), but very fast on blocks near the head.

## Database migrations

In case you encounter any issue with the database, you can force the database version using the golang-migrate command line. Please refer [here](https://github.com/golang-migrate/migrate) for more information.
More specifically, one could clean the migrations by forcing the version with <br>
`migrate -path / -database "clickhouse://host:port?username=user&password=password&database=clicks&x-multi-statement=true" force <current_version>` <br>
If specific upgrades or downgrades need to be done manually, one could do this with <br>
`migrate -path database/migration/ -database "clickhouse://host:port?username=user&password=password&database=clicks&x-multi-statement=true" -verbose up`

# Migrating from `v2` to `v3` (Postgres to Clickhouse)

During `v3.0.0` we migrated our database system from PostgreSQL to Clickhouse.
If you wish to migrate your existing database, please follow [this](https://migalabs.notion.site/PostgreSQL-to-Clickhouse-migration-611a52a457824cd494d701773365f62f) guide.

# Maintainers

@santi1234567

# Contributing

The project is open for everyone to contribute!
