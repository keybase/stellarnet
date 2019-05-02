package stellarnet

import "testing"

type resultTest struct {
	resultXDR string
	amount    string
}

var resultTests = []resultTest{
	{
		resultXDR: "AAAAAAAAAMgAAAAAAAAAAQAAAAAAAAACAAAAAAAAAAIAAAAAZzmxB0XO0YA7F1EZnKIFui/x+Zg3sSHbPkhFlZaTLOMAAAAABQrDBwAAAAFVU0QAAAAAAOimGoYeYK9g+Adz4GNG5ccsvlncrdo3YI1Y70JRHZ/cAAAAAAADj6cAAAAAAAAAAAAkJUUAAAAAR1UTezVqwN7/246Axf6IDqOe2n8fCtfUa5UBOvT5m0YAAAAABQrqnAAAAAFVU0QAAAAAAOimGoYeYK9g+Adz4GNG5ccsvlncrdo3YI1Y70JRHZ/cAAAAAAALspkAAAAAAAAAAAB2yxkAAAAAi0fpyLjYaZBgToWGyy4bJljBABuwyR/3n8CKFcoEEzUAAAABVVNEAAAAAADophqGHmCvYPgHc+BjRuXHLL5Z3K3aN2CNWO9CUR2f3AAAAAAAD0JAAAAAAA==",
		amount:    "1.0154078",
	},
	{
		resultXDR: "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAACAAAAAAAAAAEAAAAA9dCK5ZLJf6GcamYAYxpGpCCDW87noAHzmMphaxtqOxQAAAAABLefGAAAAAFVU0QAAAAAAOimGoYeYK9g+Adz4GNG5ccsvlncrdo3YI1Y70JRHZ/cAAAAAACYloAAAAAAAAAAAAScnS4AAAAAi0fpyLjYaZBgToWGyy4bJljBABuwyR/3n8CKFcoEEzUAAAABVVNEAAAAAADophqGHmCvYPgHc+BjRuXHLL5Z3K3aN2CNWO9CUR2f3AAAAAAAmJaAAAAAAA==",
		amount:    "7.7372718",
	},
}

func TestPathPaymentSourceAmount(t *testing.T) {
	for i, test := range resultTests {
		amount, err := PathPaymentSourceAmount(test.resultXDR)
		if err != nil {
			t.Errorf("test %d failed: %s", i, err)
			continue
		}
		if amount != test.amount {
			t.Errorf("test %d, amount %q, expected %q", i, amount, test.amount)
		}
	}
}
