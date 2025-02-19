package spec

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/core/types"
)

type ETH1Deposit struct {
	BlockNumber           uint64
	BlockHash             string
	TxHash                string
	LogIndex              uint64
	Sender                string
	Recipient             string
	GasUsed               uint64
	GasPrice              uint64
	DepositIndex          uint64
	ValidatorPubkey       string
	WithdrawalCredentials string
	Signature             string
	Amount                uint64
}

func (f ETH1Deposit) Type() ModelType {
	return ETH1DepositModel
}

func (f ETH1Deposit) ToArray() []interface{} {
	rows := []interface{}{
		f.BlockNumber,
		f.BlockHash,
		f.TxHash,
		f.LogIndex,
		f.Sender,
		f.Recipient,
		f.GasUsed,
		f.GasPrice,
		f.DepositIndex,
		f.ValidatorPubkey,
		f.WithdrawalCredentials,
		f.Signature,
		f.Amount,
	}
	return rows
}

func ParseETH1DepositFromLog(log *types.Log, tx *AgnosticTransaction) ETH1Deposit {
	deposit := ETH1Deposit{
		BlockNumber:           tx.BlockNumber,
		BlockHash:             log.BlockHash.String(),
		TxHash:                tx.Hash.String(),
		LogIndex:              uint64(log.Index),
		Sender:                tx.From.String(),
		Recipient:             tx.To.String(),
		GasUsed:               tx.Gas,
		GasPrice:              tx.GasPrice,
		DepositIndex:          binary.LittleEndian.Uint64(log.Data[544:552]),
		ValidatorPubkey:       "0x" + hex.EncodeToString(log.Data[192:240]),
		WithdrawalCredentials: "0x" + hex.EncodeToString(log.Data[288:320]),
		Signature:             "0x" + hex.EncodeToString(log.Data[416:512]),
		Amount:                binary.LittleEndian.Uint64(log.Data[352:360]),
	}
	return deposit
}
