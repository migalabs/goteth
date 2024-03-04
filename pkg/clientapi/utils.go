package clientapi

import (
	"strings"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	missingData = "404"
)

func response404(err string) bool {
	return strings.Contains(err, missingData)
}

func parseTxHash(hash bellatrix.Transaction) (*types.Transaction, error) {
	var parsedTx = &types.Transaction{}
	if err := parsedTx.UnmarshalBinary(hash[:]); err != nil {
		return parsedTx, err
	}
	return parsedTx, nil
}

func remove(slice []bellatrix.Transaction, s int) []bellatrix.Transaction {
	return append(slice[:s], slice[s+1:]...)
}
