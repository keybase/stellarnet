package history

func (i *accountSignersBatchInsertBuilder) Add(signer AccountSigner) error {
	return i.builder.Row(map[string]interface{}{
		"account_id": signer.Account,
		"signer":     signer.Signer,
		"weight":     signer.Weight,
	})
}

func (i *accountSignersBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
