package db

import "github.com/cortze/eth-cl-state-analyzer/pkg/db/model"

// Every database service must implement the following

type DatabaseService interface {
	init()
	Close()
	runWriters()
	Persist()
}

type WriteTask struct {
	Model model.Model
	Op    string
}
