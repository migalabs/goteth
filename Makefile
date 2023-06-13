GOCC=go
MKDIR_P=mkdir -p

BIN_PATH=./build
BIN="./build/goteth"

ifndef $(ENV_FILE)
	ENV_FILE=.env
endif

$(info ENV file: $(ENV_FILE))
include $(ENV_FILE)

.PHONY: check build install run clean

build: 
	$(GOCC) build -o $(BIN)

install:
	$(GOCC) install

run: 
	
	$(BIN) $(ANALYZER_CMD) \
		--log-level=${ANALYZER_LOG_LEVEL} \
		--bn-endpoint=${ANALYZER_BN_ENDPOINT} \
		--init-slot=${STATE_ANALYZER_INIT_SLOT} \
		--final-slot=${STATE_ANALYZER_FINAL_SLOT} \
		--db-url=${ANALYZER_DB_URL} \
		--workers-num=${STATE_ANALYZER_WORKERS_NUM} \
		--db-workers-num=${STATE_ANALYZER_DB_WORKERS_NUM} \
		--download-mode=${STATE_ANALYZER_DOWNLOAD_MODE} \
		--custom-pools=${STATE_ANALYZER_POOLS_FILE} \
		--metrics=${STATE_ANALYZER_METRICS} \
		--missing-vals=${STATE_ANALYZER_MISSING_VALS}

clean:
	rm -r $(BIN_PATH)

