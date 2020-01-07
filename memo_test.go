package stellarnet

import "testing"

type mtest struct {
	in   string
	kind string
	out  *Memo
	err  bool
}

var mtests = []mtest{
	{
		in:   "",
		kind: "NONE",
		out:  NewMemoNone(),
	},
	{
		in:   "",
		kind: "",
		out:  NewMemoNone(),
	},
	{
		in:   "public memo",
		kind: "NONE",
		err:  true,
	},
	{
		in:   "public memo",
		kind: "TEXT",
		out:  NewMemoText("public memo"),
	},
	{
		in:   "18446744073709551615",
		kind: "ID",
		out:  NewMemoID(18446744073709551615),
	},
	{
		in:   "28446744073709551615",
		kind: "ID",
		err:  true,
	},
	{
		in:   "0a0b0c0d0e0f",
		kind: "HASH",
		out:  NewMemoHash(MemoHash{10, 11, 12, 13, 14, 15}),
	},
	{
		in:   "0a0b0c0d0e0f",
		kind: "hash",
		out:  NewMemoHash(MemoHash{10, 11, 12, 13, 14, 15}),
	},
	{
		in:   "ga0b0c0d0e0f",
		kind: "HASH",
		err:  true,
	},
	{
		in:   "0a0b0c0d0e0f",
		kind: "RETURN",
		out:  NewMemoReturn(MemoHash{10, 11, 12, 13, 14, 15}),
	},
}

func TestMemoFromStrings(t *testing.T) {
	for i, test := range mtests {
		t.Logf("running test %d: %+v", i, test)
		m, err := NewMemoFromStrings(test.in, test.kind)
		if err != nil {
			if !test.err {
				t.Errorf("test %d: error %s", i, err)
			}
		} else {
			if test.err {
				t.Errorf("test %d: expected an error", i)
				continue
			}
			if test.out.String() != m.String() {
				t.Errorf("test %d: output mismatch expected %s, got %s", i, test.out.String(), m.String())
			}
		}
		t.Logf("test %d: passed", i)
	}
}
