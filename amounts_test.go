package stellarnet

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var convertXLMToOutsideUnits = []struct {
	ok   bool
	rate string
	xlm  string
	out  string
}{
	{false, "", "1", ""},
	{false, "1", "", ""},
	{false, "0", "1", ""},
	{false, "a", "1", ""},
	{false, "1e10", "1", ""},
	{false, "-1", "1", ""}, // negative exchange rate
	// XLM amount too big
	// skip negative variant because MIN_INT64 != -MAX_INT64
	{false, "2", "922337203685.4775808", "skipneg"},
	{false, "2", "0.47758071", ""}, // too many digits of precision for XLM

	{true, "1", "0", "0.0000000"},
	{true, "1", "1", "1.0000000"},
	{true, "0.5", "1", "0.5000000"},
	{true, "0.0000001", "1", "0.0000001"},
	{true, ".75", "4294967290", "3221225467.5000000"},
	{true, "2", "922337203685.4775807", "1844674407370.9551614"},
}

func TestConvertXLMToOutside(t *testing.T) {
	for i, unit := range convertXLMToOutsideUnits {
		for _, neg := range []bool{false, true} {
			t.Logf("%v: %#v", i, unit)
			s := unit.xlm
			if neg {
				s = "-" + s
			}
			y, err := ConvertXLMToOutside(s, unit.rate)
			if unit.out == "skipneg" {
				continue
			}
			require.Equal(t, unit.ok, err == nil, "converted without error: (got err:%v)", err)
			if unit.ok {
				expect := unit.out
				if neg && unit.xlm != "0" {
					expect = "-" + expect
				}
				require.Equal(t, expect, y, "converted to outside amount")
			}
		}
	}
}

var convertOutsideToXLMUnits = []struct {
	ok      bool
	rate    string
	outside string
	xlm     string
}{
	{false, "", "1", ""},
	{false, "1", "", ""},
	{false, "0", "1", ""},
	{false, "a", "1", ""},
	{false, "1e10", "1", ""},
	{false, "-1", "1", ""}, // negative exchange rate

	{true, "2", "0.47758071", "0.2387904"}, // many digits of precision are fine
	{true, "1", "0", "0.0000000"},
	{true, "1", "1", "1.0000000"},
	{true, "0.5", "1", "2.0000000"},
	{true, "0.0000001", "1", "10000000.0000000"},
	{true, ".75", "4294967290", "5726623053.3333333"},
	{true, "0.5", "922337203685.4775808", "1844674407370.9551616"}, // return can be greater than max XLM
}

func TestConvertOutsideToXLM(t *testing.T) {
	for i, unit := range convertOutsideToXLMUnits {
		for _, neg := range []bool{false, true} {
			t.Logf("%v: %#v", i, unit)
			s := unit.outside
			if neg {
				s = "-" + s
			}
			y, err := ConvertOutsideToXLM(s, unit.rate)
			require.Equal(t, unit.ok, err == nil, "converted without error: (got err:%v)", err)
			if unit.ok {
				expect := unit.xlm
				if neg && unit.outside != "0" {
					expect = "-" + expect
				}
				require.Equal(t, expect, y, "converted to xlm amount")
			}
		}
	}
}

var decimalUnits = []struct {
	ok  bool
	s   string
	val string
}{
	{false, "", ""},
	{false, ".", ""},
	{false, "-", ""},
	{false, "1-", ""},
	{false, ".1-", ""},
	{false, ".-1", ""},
	{false, "-1-", ""},
	{false, "1a", ""},
	{false, "a", ""},
	{false, "a1", ""},
	{false, "1.a", ""},
	{false, "a.1", ""},
	{false, ".1.", ""},
	{false, "1e10", ""},
	{false, "1,2", ""},
	{false, "1,", ""},
	{false, ",1", ""},
	{false, "1/2", ""},
	{false, "1b10", ""},
	{false, " 10.95", ""},
	{false, "10.95 ", ""},
	{false, "10. 95 ", ""},
	{false, "1 0.95 ", ""},
	{false, "10.9 5", ""},
	{false, "--10.95", ""},

	{true, "1", "1/1"},
	{true, "1.", ""},
	{true, ".1", "1/10"},
	{true, "1.1", ""},

	{true, "3333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333", ""},
	{true, "3333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333.", ""},
	{true, ".3333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333", ""},
	{true, "3333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333.3333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333", "33333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333333/10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},

	{true, "10.95", "219/20"},
	{true, "1234567", "1234567/1"},
	{true, "1234567.8910", ""},
	{true, "005.00500", ""},
}

func TestDecimalStrictRegex(t *testing.T) {
	for i, unit := range decimalUnits {
		for _, neg := range []bool{false, true} {
			t.Logf("%v: %#v", i, unit)
			s := unit.s
			if neg {
				s = "-" + s
			}
			require.Equal(t, unit.ok, decimalStrictRE.MatchString(s))
		}
	}
}

func TestParseAmount(t *testing.T) {
	for i, unit := range decimalUnits {
		for _, neg := range []bool{false, true} {
			t.Logf("%v: %#v", i, unit)
			s := unit.s
			if neg {
				s = "-" + s
			}
			v, err := ParseAmount(s)
			t.Logf("-> (%v, %v)", v, err)
			require.Equal(t, unit.ok, err == nil, "parsed without error")
			if unit.ok {
				if unit.val != "" {
					if neg {
						require.Equal(t, "-"+unit.val, v.String())
					} else {
						require.Equal(t, unit.val, v.String())
					}
				}
			}
		}
	}
}

var withinUnits = []struct {
	a1, a2, f  string
	ok, answer bool
}{
	{"", "1", ".5", false, false},
	{"1", "", ".5", false, false},
	{"1", "1", "", false, false},
	{"1", "1", "-.5", false, false},

	{"100", "110", ".1", true, true},
	{"3000", "6500", ".5", true, false},
	{"100", "105", ".01", true, false},
	{"100", "90", ".1", true, true},
	{"100", "90", ".09999", true, false},
	{"192329", "190405.71", ".01", true, true},
	{"192329", "194300", ".01", true, false},
	{"0", "0", "2", true, true},
	{"0", "0.001", "2", true, false},
	{"0.0001", "0", "2", true, false},
	{"12.5", "12.5", "0", true, true},
	{"1", "-1", "2", true, true},
	{"1", "-1", "1", true, false},
	{"1", "-1.000001", ".9", true, false},
}

func TestWithinFactorStellarAmounts(t *testing.T) {
	for i, unit := range withinUnits {
		t.Logf("%v: %#v", i, unit)
		within, err := WithinFactorStellarAmounts(unit.a1, unit.a2, unit.f)
		t.Logf("-> (%v, %v)", within, err)
		require.Equal(t, unit.ok, err == nil, "ran without error")
		require.Equal(t, unit.answer, within, "answer to within")
	}
}

var StellarAmountParsingTests = []struct {
	S     string
	I     int64
	valid bool
}{
	{"100.0000000", 1000000000, true},
	{"-100.0000000", -1000000000, true},
	{"100.0000001", 1000000001, true},
	{"123.0000001", 1230000001, true},
	{"123.00000001", 0, false},
	{"922337203685.4775807", 9223372036854775807, true},
	{"922337203685.4775808", 0, false},
	{"922337203686", 0, false},
	{"-922337203685.4775808", -9223372036854775808, true},
	{"-922337203685.4775809", 0, false},
	{"-922337203686", 0, false},
	{"1000000000000.0000000", 0, false},
	{"1000000000000", 0, false},
	{"-0.5000000", -5000000, true},
	{"0.5000000", 5000000, true},
	{"0.12345678", 0, false},
	{"1/2", 0, false},
	{"2e3", 0, false},
	{"2E3", 0, false},
	// Expensive inputs:
	{strings.Repeat("1", 1000000), 0, false},
	{"1E9223372036854775807", 0, false},
	{"1e9223372036854775807", 0, false},
	{"Inf", 0, false},
}

func TestParseStellarAmount(t *testing.T) {
	for _, v := range StellarAmountParsingTests {
		o, err := ParseStellarAmount(v.S)
		if v.valid {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
		require.Equal(t, o, v.I, "%s parsed to %d, not %d", v.S, o, v.I)
	}
}

func TestStringFromStellarAmount(t *testing.T) {
	for _, v := range StellarAmountParsingTests {
		if !v.valid {
			continue
		}

		o := StringFromStellarAmount(v.I)
		require.Equal(t, o, v.S, "%d stringified to %s, not %s", v.I, o, v.S)
	}
}

func TestPathPaymentMaxValue(t *testing.T) {
	max, err := PathPaymentMaxValue("100")
	require.NoError(t, err)
	require.Equal(t, "105.0000000", max)

	max, err = PathPaymentMaxValue("2.000")
	require.NoError(t, err)
	require.Equal(t, "2.1000000", max)

	max, err = PathPaymentMaxValue("1234567.8989872")
	require.NoError(t, err)
	require.Equal(t, "1296296.2939365", max)
}

type feeTest struct {
	in  int32
	out string
}

var feeTests = []feeTest{
	{in: 100, out: "0.0000100"},
	{in: 1500, out: "0.0001500"},
	{in: 1234, out: "0.0001234"},
}

func TestFeeString(t *testing.T) {
	for i, test := range feeTests {
		if FeeString(test.in) != test.out {
			t.Errorf("%d: FeeString(%d) => %s != %s", i, test.in, FeeString(test.in), test.out)
		}
	}
}

func TestGetStellarExchangeRate(t *testing.T) {
	rate, err := GetStellarExchangeRate("1", "1")
	require.NoError(t, err)
	require.Equal(t, "1.0000000", rate)

	rate, err = GetStellarExchangeRate("1", "10")
	require.NoError(t, err)
	require.Equal(t, "10.0000000", rate)

	rate, err = GetStellarExchangeRate("10", "1")
	require.NoError(t, err)
	require.Equal(t, "0.1000000", rate)

	rate, err = GetStellarExchangeRate("0", "0.1384476")
	require.Error(t, err)

	rate, err = GetStellarExchangeRate("15.23", "0")
	require.Equal(t, "0.0000000", rate)

	rate, err = GetStellarExchangeRate("0.0002122", "0.0001000")
	require.NoError(t, err)
	require.Equal(t, "0.4712535", rate)
}
