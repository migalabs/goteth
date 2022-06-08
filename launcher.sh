#!/bin/bash

CLI_NAME="state-analyzer"

echo "launching State-Analyzer"


BN_ENDPOINT="https://20PdJoS82pnejJJ9joDMnbjsQ32:0c9b868d8621332ea91c7fc24c5fc34f@eth2-beacon-mainnet.infura.io"
#BN_ENDPOINT="http://localhost:5052"
OUT_FOLDER="results"
INIT_SLOT="300000"
FINAL_SLOT="300063"
VALIDATOR_LIST_FILE="test_validators.json"

go get
go build -o $CLI_NAME



"./$CLI_NAME" rewards --log-level=$1 --bn-endpoint="$BN_ENDPOINT" --outfolder="$OUT_FOLDER" --init-slot="$INIT_SLOT" --final-slot="$FINAL_SLOT" --validator-indexes="$VALIDATOR_LIST_FILE"


