# GotEth

GotEth is a go-written client that indexes all validator-related duties, parameters and transactions from Ethereum's consensus and execution layers.

The client indexes all the validator/epoch related metrics into a set of clickhouse tables which later on can be used to monitor the performance of validators in the beaconchain. See the [docs/tables.md](https://github.com/migalabs/goteth/blob/master/docs/tables.md) for more information on the tables indexed by Goteth.

This tool has been used to power the

- [pandametrics.xyz](https://pandametrics.xyz/) public dashboard
- [ethseer.io](https://ethseer.io) block explorer
- [monitoreth.io](https://monitoreth.io/nodes#validators-entities) public dashboard

## Prerequisites

To use the tool, the following requirements need to be installed in the machine:

- [go](https://go.dev/doc/install) preferably on its 1.21 version or above. Go also needs to be executable from the terminal.
- Clickhouse DB
- Access to an Ethereum consensus archival node (we have only tested using lighthouse in archival mode, other clients/configs might not work). IMPORTANT: Goteth requires the `/eth/v2/debug/beacon/states` endpoint enabled. To be able to fetch blob sidecars, the `--supernode` flag must be enabled in Lighthouse after Fulu hardfork.
- Access to an Ethereum execution node (optional)
- Access to a Clickhouse server database (use native port, usually 9000)

## Cloning

Goteth uses a fork of [github.com/attestantio/go-relay-client](https://github.com/attestantio/go-relay-client) as a git submodule. In order to be able to run goteth, you will need to clone the submodule as well with: `--recurse-submodules` flag.

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
- api_rewards: block rewards (consensus layer) are hard to calculate, but they can be downloaded from the Beacon API. However, keep in mind this takes a few seconds per block when not at the head (not recommended for backfilling). Without this, reward cannot be compared to max_reward when a validator is a proposer (32/1000k validators in an epoch). It depends on the Lighthouse API and we have registered some cases where the block reward was not returned.
- transactions: requests transaction receipts from the execution layer, eth1 deposits and blob sidecars from consensus layer (activates block metrics)

Go to [docs/tables.md](https://github.com/migalabs/goteth/blob/master/docs/tables.md) for more information on the tables indexed by Goteth.

### Table Sizes

Data from mainnet, may 2025.

- `t_validator_rewards_summary`: 1 month of data: `68GB`
- `t_validator_rewards_aggregations`: The size of `t_validator_rewards_summary` divided by `GOTETH_REWARDS_AGGREGATION_EPOCHS`. For example, if you set `GOTETH_REWARDS_AGGREGATION_EPOCHS=10`, the size of this table will be `6.8GB`.
- `t_transactions`: Since merge: `405GB`
- Rest of tables: `10GB`

Most tables in Goteth use the ClickHouse `ReplacingMergeTree` engine. For optimal operation, ClickHouse requires free disk space equal to the full size of each table to perform background optimizations and deletions. For example, if the rewards table occupies 68GB, you must have at least 68GB of free disk space available to safely delete rows (such as when using the validator window script). Insufficient free space may prevent these operations from completing successfully, further complicating the situation.

### Validator rewards aggregation

The tool can aggregate the rewards of the validators in the `t_validator_rewards_summary` table. This is done by aggregating the rewards of the last `GOTETH_REWARDS_AGGREGATION_EPOCHS` epochs. The aggregation is done by summing up the columns of each validator in the last `GOTETH_REWARDS_AGGREGATION_EPOCHS` epochs and storing the result in the `t_validator_rewards_aggregations` table.

It can be very useful when monitoring rewards over a long period of time, without having to worry about the size of the `t_validator_rewards_summary` table, if combined with the [`val-window` command](#validator-rewards-window). Please note that `GOTETH_REWARDS_AGGREGATION_EPOCHS` must be set to a value greater than 1 to be enabled and also be lower than `GOTETH_VAL_WINDOW_NUM_EPOCHS` to avoid data loss.

## Download mode

- Historical: this mode loops over slots between `initSlot` and `finalSlot`, which are configurable. Once all slots have been analyzed, the tool finishes the execution.
- Finalized: `initSlot` and `finalSlot` are ignored. The tool starts the historical mode from the database last slot to the current head (beacon node) and then follows the chain head. To do this, the tool subscribes to `head` events. See [here](https://ethereum.github.io/beacon-APIs/#/Events/eventstream) for more information.

## Running the tool

To execute the tool, you can simply modify the `.env` file with your own configuration.

_Running the tool (configurable in the `.env` file)_:

```
docker compose up goteth
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
   --download-mode value   example: historical,finalized. Default: finalized
   --metrics value         example: epoch,block,rewards,transactions,api_rewards,blob_sidecars. Empty for all (default: epoch,block)
   --prometheus-port value Port on which to expose prometheus metrics (default: 9081)
   --max-request-retries value         Number of retries to make when a request fails. For head mode it shouldn't be higher than 3-4, for historical its recommended to be higher (default: 3)
   --beacon-contract-address value     Beacon contract address. Can be 'mainnet', 'holesky', 'sepolia' or directly the contract address in format '0x...' (default: mainnet)
   --help, -h              show help (default: false)
```

### Validator Rewards Window

The validator rewards table can get large in the database (see [Table Sizes](#table-sizes)), storing rewards for epochs which might not be relevant anymore to the user. We have developed a subcommand of the tool which maintains the last n epochs of rewards data in the database, prunning from the defined threshold backwards. So, one can configure the tool to maintain the last 100 epochs of data in the database, while prunning the rest.
The window only affects the `t_validator_rewards_summary` table.

Simply configure `GOTETH_VAL_WINDOW_NUM_EPOCHS` variable and run

```
docker compose up val-window
```

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
