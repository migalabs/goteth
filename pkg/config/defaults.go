package config

var (
	DefaultLogLevel                 string = "info"
	DefaultInitSlot                 int    = 0
	DefaultFinalSlot                int    = 0
	DefaultBnEndpoint               string = ""
	DefaultElEndpoint               string = ""
	DefaultRewardsAggregationEpochs int    = 1
	DefaultDBUrl                    string = "clickhouse://username:password@localhost:9000/goteth?x-multi-statement=true&max_memory_usage=10000000000"
	DefaultDownloadMode             string = "finalized"
	DefaultWorkerNum                int    = 4
	DefaultDbWorkerNum              int    = 4
	DefaultMetrics                  string = "epoch,block"
	DefaultPrometheusPort           int    = 9080
	DefaultValidatorWindowEpochs    int    = 100
	DefaultMaxRequestRetries        int    = 3
	DefaultBeaconContractAddress    string = "mainnet"
	DefaultTransactionGapBatchSize  int    = 1000
	DefaultTransactionGapWorkers    int    = 4
)
