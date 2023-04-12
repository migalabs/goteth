package model

var (
	INSERT_OP = "INSERT"
	UPDATE_OP = "UPDATE"
	DROP_OP   = "DROP"
)

type Model interface { // simply to enforce a Model interface
	InsertOp() bool // whether insert is activated for this model
	DropOp() bool   // whether drop is activated for this model
}
