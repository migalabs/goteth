#!make

GOCC=go
MKDIR_P=mkdir -p

BIN_PATH=./build
BIN="./build/eth2-state-analyzer"

include .env

.PHONY: check build install run clean

build: 
	$(GOCC) build -o $(BIN)

install:
	$(GOCC) install

run: 
	$(BIN) $(STATE_ANALYZER_CMD) \
        --log-level=${STATE_ANALYZER_LOG_LEVEL} \
        --bn-endpoint=${STATE_ANALYZER_BN_ENDPOINT} \
        --outfolder=${STATE_ANALYZER_OUTFOLDER} \
        --init-slot=${STATE_ANALYZER_INIT_SLOT} \
        --final-slot=${STATE_ANALYZER_FINAL_SLOT} \
        --validator-indexes=${STATE_ANALYZER_VALIDATOR_INDEXES} \
        --db-url=${STATE_ANALYZER_DB_URL} \
        --workers-num=${STATE_ANALYZER_WORKERS_NUM} \
        --db-workers-num=${STATE_ANALYZER_DB_WORKERS_NUM}

clean:
	rm -r $(BIN_PATH)

