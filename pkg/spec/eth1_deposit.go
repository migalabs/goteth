package spec

type ETH1Deposit struct {
	BlockNumber           uint64
	BlockHash             string
	BlockTimestamp        uint64
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
		f.BlockTimestamp,
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
