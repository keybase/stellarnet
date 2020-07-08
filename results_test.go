package stellarnet

import (
	"strings"
	"testing"
)

type resultTest struct {
	resultXDR string
	opIndex   int
	amount    string
}

var resultTests = []resultTest{
	{
		resultXDR: "AAAAAAAAAMgAAAAAAAAAAQAAAAAAAAACAAAAAAAAAAIAAAAAZzmxB0XO0YA7F1EZnKIFui/x+Zg3sSHbPkhFlZaTLOMAAAAABQrDBwAAAAFVU0QAAAAAAOimGoYeYK9g+Adz4GNG5ccsvlncrdo3YI1Y70JRHZ/cAAAAAAADj6cAAAAAAAAAAAAkJUUAAAAAR1UTezVqwN7/246Axf6IDqOe2n8fCtfUa5UBOvT5m0YAAAAABQrqnAAAAAFVU0QAAAAAAOimGoYeYK9g+Adz4GNG5ccsvlncrdo3YI1Y70JRHZ/cAAAAAAALspkAAAAAAAAAAAB2yxkAAAAAi0fpyLjYaZBgToWGyy4bJljBABuwyR/3n8CKFcoEEzUAAAABVVNEAAAAAADophqGHmCvYPgHc+BjRuXHLL5Z3K3aN2CNWO9CUR2f3AAAAAAAD0JAAAAAAA==",
		opIndex:   0,
		amount:    "1.0154078",
	},
	{
		resultXDR: "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAACAAAAAAAAAAEAAAAA9dCK5ZLJf6GcamYAYxpGpCCDW87noAHzmMphaxtqOxQAAAAABLefGAAAAAFVU0QAAAAAAOimGoYeYK9g+Adz4GNG5ccsvlncrdo3YI1Y70JRHZ/cAAAAAACYloAAAAAAAAAAAAScnS4AAAAAi0fpyLjYaZBgToWGyy4bJljBABuwyR/3n8CKFcoEEzUAAAABVVNEAAAAAADophqGHmCvYPgHc+BjRuXHLL5Z3K3aN2CNWO9CUR2f3AAAAAAAmJaAAAAAAA==",
		opIndex:   0,
		amount:    "7.7372718",
	},
	{
		resultXDR: "AAAAAAAAAMgAAAAAAAAAAgAAAAAAAAACAAAAAAAAAAEAAAAA850DFsVvWOIBUnfhUQulcmwJLB216tQgkV9W7yGEV0EAAAAABfJERQAAAAFVU0QAAAAAAOimGoYeYK9g+Adz4GNG5ccsvlncrdo3YI1Y70JRHZ/cAAAAAACYloAAAAAAAAAAAAWtG4QAAAAAQNAe5MSQIvC9d7lUWkos/Aiup+UsCdrQMTy/VUrPlKsAAAABVVNEAAAAAADophqGHmCvYPgHc+BjRuXHLL5Z3K3aN2CNWO9CUR2f3AAAAAAAmJaAAAAAAAAAAAUAAAAAAAAAAA==",
		opIndex:   0,
		amount:    "9.5230852",
	},
}

func TestPathPaymentSourceAmount(t *testing.T) {
	for i, test := range resultTests {
		amount, err := PathPaymentSourceAmount(test.resultXDR, test.opIndex)
		if err != nil {
			t.Errorf("test %d failed: %s", i, err)
			continue
		}
		if amount != test.amount {
			t.Errorf("test %d, amount %q, expected %q", i, amount, test.amount)
		}
	}
}

type pathTest struct {
	envelopeXDR string
	opIndex     int
	path        string
}

var pathTests = []pathTest{
	{
		envelopeXDR: "AAAAAMLhohLPFgPhIDVGCaCUaV4ADNq/J73lF/UduzXPIa9fAAAAyAFeEhsAAAAaAAAAAAAAAAAAAAACAAAAAAAAAAIAAAAAAAAAAAaOd4AAAAAAQNAe5MSQIvC9d7lUWkos/Aiup+UsCdrQMTy/VUrPlKsAAAABVVNEAAAAAADophqGHmCvYPgHc+BjRuXHLL5Z3K3aN2CNWO9CUR2f3AAAAAAAmJaAAAAAAgAAAAAAAAABVVNEAAAAAADophqGHmCvYPgHc+BjRuXHLL5Z3K3aN2CNWO9CUR2f3AAAAAAAAAAFAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAALZmVkLm5ldHdvcmsAAAAAAAAAAAAAAAABzyGvXwAAAEBU5nfxCaqD+uYKfUUwDbZ3RwAWTrp65e5NdpnDRpl4ZCJ6a07E4qDE0Z/mDLk7piWB8UqPkefk0ylfVJOuMMEO",
		opIndex:     0,
		path:        "XLM -> USD/GDUKMGUGDZQK6YHYA5Z6AY2G4XDSZPSZ3SW5UN3ARVMO6QSRDWP5YLEX",
	},
}

func TestPathPaymentIntermediatePath(t *testing.T) {
	for i, test := range pathTests {
		path, err := PathPaymentIntermediatePath(test.envelopeXDR, test.opIndex)
		if err != nil {
			t.Errorf("test %d failed: %s", i, err)
			continue
		}

		pathSummary := make([]string, len(path))
		for i, p := range path {
			pathSummary[i] = AssetBaseSummary(p)
		}
		single := strings.Join(pathSummary, " -> ")
		t.Logf("path: %s\n", single)
		if single != test.path {
			t.Errorf("test %d, path %q, expected %q", i, single, test.path)
		}
	}
}
