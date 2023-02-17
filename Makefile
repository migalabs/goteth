#!make

GOCC=go
MKDIR_P=mkdir -p

BIN_PATH=./build
BIN="./build/eth-cl-state-analyzer"



.PHONY: check build install run clean

build: 
	$(GOCC) build -o $(BIN)

install:
	$(GOCC) install


ifndef env_file
env_file=.env # default
endif

include $(env_file)
ifeq ($(STATE_ANALYZER_CMD),"rewards")
run: 
		$(BIN) $(STATE_ANALYZER_CMD) \
			--log-level=${STATE_ANALYZER_LOG_LEVEL} \
			--bn-endpoint=${STATE_ANALYZER_BN_ENDPOINT} \
			--init-slot=${STATE_ANALYZER_INIT_SLOT} \
			--final-slot=${STATE_ANALYZER_FINAL_SLOT} \
			--db-url=${STATE_ANALYZER_DB_URL} \
			--workers-num=${STATE_ANALYZER_WORKERS_NUM} \
			--db-workers-num=${STATE_ANALYZER_DB_WORKERS_NUM} \
			--download-mode=${STATE_ANALYZER_DOWNLOAD_MODE} \
			--custom-pools=${STATE_ANALYZER_POOLS_FILE} \
			--metrics=${STATE_ANALYZER_METRICS}
endif

ifeq ($(STATE_ANALYZER_CMD),"blocks")
run: 
		$(BIN) $(STATE_ANALYZER_CMD) \
			--log-level=${STATE_ANALYZER_LOG_LEVEL} \
			--bn-endpoint=${STATE_ANALYZER_BN_ENDPOINT} \
			--init-slot=${STATE_ANALYZER_INIT_SLOT} \
			--final-slot=${STATE_ANALYZER_FINAL_SLOT} \
			--db-url=${STATE_ANALYZER_DB_URL} \
			--workers-num=${STATE_ANALYZER_WORKERS_NUM} \
			--db-workers-num=${STATE_ANALYZER_DB_WORKERS_NUM} \
			--download-mode=${STATE_ANALYZER_DOWNLOAD_MODE}
endif

clean:
	rm -r $(BIN_PATH)

