package config

var (
	DefaultLogLevel                 string = "info"
	DefaultInitSlot                 int    = 0
	DefaultFinalSlot                int    = 0
	DefaultBnEndpoint               string = ""
	DefaultElEndpoint               string = ""
	DefaultRewardsAggregationEpochs int    = 1
	DefaultDBUrl                    string = "postgres://user:password@localhost:5432/goteth"
	DefaultDownloadMode             string = "finalized"
	DefaultWorkerNum                int    = 4
	DefaultDbWorkerNum              int    = 4
	DefaultMetrics                  string = "epoch,block"
	DefaultPrometheusPort           int    = 9080
	DefaultValidatorWindowEpochs    int    = 100
	DefaultMaxRequestRetries        int    = 3
	DefaultBeaconContractAddress    string = "mainnet"
)
