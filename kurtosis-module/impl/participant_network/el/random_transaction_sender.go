package el

type RandomTransactionSender interface {
	SendRandomTransaction() error
}