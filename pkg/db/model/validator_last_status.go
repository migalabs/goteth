package model

import "github.com/attestantio/go-eth2-client/spec/phase0"

var (
	ValidatorLastStatusModelOps = map[string]bool{
		INSERT_OP: true,
		DROP_OP:   false,
	}
)

type ValidatorLastStatus struct {
	ValIdx         phase0.ValidatorIndex
	Epoch          phase0.Epoch
	CurrentBalance phase0.Gwei
	CurrentStatus  int
}

func (f ValidatorLastStatus) InsertOp() bool {
	return true
}

func (f ValidatorLastStatus) DropOp() bool {
	return false
}
