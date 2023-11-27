# GotEth

GotEth is a go-written client that indexes all validator-related duties and parameters from Ethereum's beaconchain by fetching the CL States from a node (preferable a locally running archival node).

The client indexes all the validator/epoch related metrics into a set of postgreSQL tables which later on can be used to monitor the performance of validators in the beaconchain.

This tool has been used to power the 
- [pandametrics.xyz](https://pandametrics.xyz/) public dashboard
- [ethseer.io](https://ethseer.io) public dashboard

### Prerequisites
To use the tool, the following requirements need to be installed in the machine:
- [go](https://go.dev/doc/install) preferably on its 1.19 version or above. Go also needs to be executable from the terminal.
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

### Metrics: database tables

- block: downloads withdrawals, blocks and block rewards
- epoch: download epoch metrics, proposer duties, validator last status,
- rewards: persists validator rewards metrics to database (activates epoch metrics)
- api_rewards (EXPERIMENTAL): block rewards are hard to calculate, but they can be downloaded from the Beacon API. However, keep in mind this takes a few seconds per block when not at the head. Without this, reward cannot be compared to max_reward when a validator is a proposer (32/900K validators in an epoch). It depends on the Lighthouse API and we have registered some cases where the block reward was not returned.
- transactions: requests transaction receipts from the execution layer (activates block metrics)

### Running the tool
To execute the tool, you can simply modify the `.env` file with your own configuration.

*Running the tool (configurable in the `.env` file)*:
```
docker-compose up
```

*Available Commands*:
```
COMMANDS:
   blocks   analyze the Beacon Block of a given slot range
   help, h  Shows a list of commands or help for one command
```

*Available Options (configurable in the `.env` file)*
```

Blocks
OPTIONS:
   --bn-endpoint value     beacon node endpoint (to request the Beacon Blocks)
   --el-endpoint value 	   execution node endpoint (to request the Transaction Receipts, optional)
   --init-slot value       init slot from where to start (default: 0)
   --final-slot value      init slot from where to finish (default: 0)
   --log-level value       log level: debug, warn, info, error
   --db-url value          example: postgresql://beaconchain:beaconchain@localhost:5432/beacon_states
   --workers-num value     example: 3 (default: 4)
   --db-workers-num value  example: 3 (default: 4)
   --download-mode value   example: hybrid,historical,finalized. Default: hybrid
   --metrics value         example: epoch,block,rewards,transactions. Empty for all (default: epoch,block)
   --prometheus-port value Port on which to expose prometheus metrics (default: 9081)
   --help, -h              show help (default: false)
```

# Notes

Validator metrics consume 95% of the database size. Please bear in mind that for 1k epochs of data, validator metrics consume around 100GB and CPU load will also increase. Exporting validator metrics has been tested using LH archival (states every 32 slots) and 32 core (AMD Ryzen 9500X) 128GB RAM machine.
The tool will export the metrics but it might take some time if the machine is not as powerful.

Keep in mind rewards data also downloads block rewards from the Beacon API. This is very slow on historical blocks (3 seconds per block), but very fast on blocks near the head.

## Database migrations

In case you encounter any issue with the database, you can force the database version using the golang-migrate command line. Please refer [here](https://github.com/golang-migrate/migrate) for more information.
More specifically, one could clean the migrations by forcing the version with <br>
```migrate -path / -database "postgresql://username:secretkey@localhost:5432/database_name?sslmode=disable" force <current_version>``` <br>
If specific upgrades or downgrades need to be done manually, one could do this with <br>
```migrate -path database/migration/ -database "postgresql://username:secretkey@localhost:5432/database_name?sslmode=disable" -verbose up```


# Maintainers
@cortze @tdahar

# Contributing
The project is open for everyone to contribute! 