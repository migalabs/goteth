package clientapi

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// GetReceipt retrieves receipt for the given transaction hash
func GetReceipt(txHash common.Hash, client *APIClient) (*types.Receipt, error) {
	receipt, err := client.ELApi.TransactionReceipt(client.ctx, txHash)
	return receipt, err
}
