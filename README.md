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
   blocks   analyze the Beacon Block of a given slot range
   help, h  Shows a list of commands or help for one command
```

*Available Options (configurable in the `.env` file)*
```
Rewards
OPTIONS:
   --bn-endpoint value        beacon node endpoint (to request the BeaconStates)
   --init-slot value          init slot from where to start (default: 0)
   --final-slot value         init slot from where to finish (default: 0)
   --log-level value          log level: debug, warn, info, error
   --db-url value             example: postgresql://beaconchain:beaconchain@localhost:5432/beacon_states
   --workers-num value        example: 50 (default: 0)
   --db-workers-num value     example: 50 (default: 0)
   --custom-pools value       example: pools.csv. Columns: f_val_idx,pool_name (Experimental)
   --metrics value            example: epoch,validator, epoch. Empty for all (default: epoch)
   --help, -h                 show help (default: false)

Blocks
OPTIONS:
   --bn-endpoint value     beacon node endpoint (to request the Beacon Blocks)
   --el-endpoint value 	   execution node endpoint (to request the Transaction Receipts, optional)
   --init-slot value       init slot from where to start (default: 0)
   --final-slot value      init slot from where to finish (default: 0)
   --log-level value       log level: debug, warn, info, error
   --db-url value          example: postgresql://beaconchain:beaconchain@localhost:5432/beacon_states
   --workers-num value     example: 50 (default: 0)
   --db-workers-num value  example: 50 (default: 0)
   --download-mode value   example: hybrid,historical,finalized. Default: hybrid
   --help, -h              show help (default: false)
```

Additionally, you may run using the docker-compose file (see list of services in docker-compose file):
```
docker-compose up
```

# Notes

Validator metrics consume 95% of the database size. Please bear in mind that for 1k epochs of data, validator metrics consume around 100GB and CPU load will also increase. Exporting validator metrics has been tested using LH archival (states every 32 slots) and 32 core (AMD Ryzen 9500X) 128GB RAM machine.
The tool will export the metrics but it might take some time if the machine is not as powerful.

If no pools file is input to the tool, missing-vals flag is useless. If you want to track all validators under a unique pool go get pool statistics then you can add a single validator in the pools csv and add the missing-vals flag, this should put all validators in a single pool called "others". Please bear in mind this functionality is experimental and has not been tested yet.

## Database migrations

In case you encounter any issue with the database, you can force the database version using the golang-migrate command line. Please refer [here](https://github.com/golang-migrate/migrate) for more information.
More specifically, one could clean the migrations by forcing the version with <br>
```migrate force -database "postgresql://username:secretkey@localhost:5432/database_name?sslmode=disable" <current_version>``` <br>
If specific upgrades or downgrades need to be done manually, one could do this with <br>
```migrate -path database/migration/ -database "postgresql://username:secretkey@localhost:5432/database_name?sslmode=disable" -verbose up```


# Maintainers
@cortze , @tdahar

# Contributing
The project is open for everyone to contribute! 