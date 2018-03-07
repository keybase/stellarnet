package stellarnet

type Transaction struct {
	Internal   TransactionEmbed
	Operations []Operation
}
