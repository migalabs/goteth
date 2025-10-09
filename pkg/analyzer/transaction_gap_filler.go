package analyzer

import (
	"context"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/migalabs/goteth/pkg/config"
	"github.com/migalabs/goteth/pkg/db"
	"github.com/migalabs/goteth/pkg/spec"
)

type TransactionGapFiller struct {
	analyzer *ChainAnalyzer
}

func NewTransactionGapFiller(ctx context.Context, cfg config.TransactionGapConfig) (*TransactionGapFiller, error) {
	analyzerCfg := config.NewAnalyzerConfig()
	analyzerCfg.LogLevel = cfg.LogLevel
	analyzerCfg.BnEndpoint = cfg.BnEndpoint
	analyzerCfg.ElEndpoint = cfg.ElEndpoint
	analyzerCfg.DBUrl = cfg.DBUrl
	analyzerCfg.MaxRequestRetries = cfg.MaxRequestRetries
	analyzerCfg.BeaconContractAddress = cfg.BeaconContractAddress
	analyzerCfg.Metrics = cfg.Metrics
	analyzerCfg.DownloadMode = "finalized"

	chainAnalyzer, err := NewChainAnalyzer(ctx, *analyzerCfg)
	if err != nil {
		return nil, err
	}

	return &TransactionGapFiller{analyzer: chainAnalyzer}, nil
}

func (f *TransactionGapFiller) Close() {
	if f == nil || f.analyzer == nil {
		return
	}
	f.analyzer.dbClient.Finish()
	f.analyzer.cancel()
}

func (f *TransactionGapFiller) FindGapsRange(startSlot, endSlot uint64) ([]db.TransactionGap, error) {
	if f == nil || f.analyzer == nil {
		return nil, fmt.Errorf("transaction gap filler not initialized")
	}
	return f.analyzer.dbClient.RetrieveTransactionGapsRange(startSlot, endSlot)
}

func (f *TransactionGapFiller) LastSlot() (phase0.Slot, error) {
	if f == nil || f.analyzer == nil {
		return 0, fmt.Errorf("transaction gap filler not initialized")
	}
	return f.analyzer.dbClient.RetrieveLastSlot()
}

func (f *TransactionGapFiller) FindMissingBlockMetricsRange(startSlot, endSlot uint64) ([]uint64, error) {
	if f == nil || f.analyzer == nil {
		return nil, fmt.Errorf("transaction gap filler not initialized")
	}
	return f.analyzer.dbClient.RetrieveMissingBlockMetricsRange(startSlot, endSlot)
}

func (f *TransactionGapFiller) ReprocessSlot(slot phase0.Slot) error {
	if f == nil || f.analyzer == nil {
		return fmt.Errorf("transaction gap filler not initialized")
	}

	block, err := f.analyzer.cli.RequestBeaconBlock(slot)
	if err != nil {
		return fmt.Errorf("requesting block at slot %d: %w", slot, err)
	}
	if block == nil {
		return fmt.Errorf("beacon node returned nil block at slot %d", slot)
	}

	if f.analyzer.metrics.Block {
		if err := f.analyzer.dbClient.PersistBlocks([]spec.AgnosticBlock{*block}); err != nil {
			return fmt.Errorf("persisting block metrics at slot %d: %w", slot, err)
		}
	}

	f.analyzer.processWithdrawals(block)

	if f.analyzer.metrics.Transactions {
		f.analyzer.ProcessETH1Data(block)
	} else {
		log.WithField("slot", slot).Warn("transactions metric disabled, skipping transaction gap repair")
	}

	f.analyzer.processBLSToExecutionChanges(block)
	f.analyzer.processDeposits(block)

	return nil
}
