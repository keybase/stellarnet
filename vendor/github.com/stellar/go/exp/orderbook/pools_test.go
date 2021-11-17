package orderbook

import (
	"math"
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestLiquidityPoolExchanges(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		for _, asset := range []xdr.Asset{usdAsset, eurAsset} {
			payout, err := makeTrade(eurUsdLiquidityPool, asset, tradeTypeDeposit, 500)
			assert.NoError(t, err)
			assert.EqualValues(t, 332, int64(payout))
			// reserves would now be: 668 of A, 1500 of B
			// note pool object is unchanged so looping is safe
		}

		for _, asset := range []xdr.Asset{usdAsset, eurAsset} {
			payout, err := makeTrade(eurUsdLiquidityPool, asset, tradeTypeExpectation, 332)
			assert.NoError(t, err)
			assert.EqualValues(t, 499, int64(payout))
		}

		// More sanity checks; if they fail, something was changed about how
		// constant product liquidity pools work.
		//
		// We use these oddly-specific values because we rely on them again to
		// validate paths in later tests.
		testTable := []struct {
			dstAsset       xdr.Asset
			pool           xdr.LiquidityPoolEntry
			expectedPayout xdr.Int64
			expectedInput  xdr.Int64
		}{
			{yenAsset, eurYenLiquidityPool, 100, 112},
			{eurAsset, eurUsdLiquidityPool, 112, 127},
			{nativeAsset, nativeUsdPool, 5, 25},
			{usdAsset, usdChfLiquidityPool, 127, 342},
			{usdAsset, usdChfLiquidityPool, 27, 58},
		}

		for _, test := range testTable {
			needed, err := makeTrade(test.pool, test.dstAsset,
				tradeTypeExpectation, test.expectedPayout)

			assert.NoError(t, err)
			assert.EqualValuesf(t, test.expectedInput, needed,
				"expected exchange of %d %s -> %d %s, got %d",
				test.expectedInput, getCode(getOtherAsset(test.dstAsset, test.pool)),
				test.expectedPayout, getCode(test.dstAsset),
				needed)
		}
	})

	t.Run("fail on bad exchange amounts", func(t *testing.T) {
		badValues := []xdr.Int64{math.MaxInt64, math.MaxInt64 - 99, 0, -100}
		for _, badValue := range badValues {
			_, err := makeTrade(eurUsdLiquidityPool, usdAsset, tradeTypeDeposit, badValue)
			assert.Error(t, err)
		}
	})
}

// TestLiquidityPoolMath is a robust suite of tests to ensure that theliquidity
// pool calculation functions are correct, taken from Stellar Core's tests here:
//
// https://github.com/stellar/stellar-core/blob/master/src/transactions/test/ExchangeTests.cpp#L948
func TestLiquidityPoolMath(t *testing.T) {
	iMax := xdr.Int64(math.MaxInt64)
	send, recv := tradeTypeDeposit, tradeTypeExpectation

	t.Run("deposit edge cases", func(t *testing.T) {
		// Sending deposit would overflow the reserve:
		//   low reserves but high deposit
		assertPoolExchange(t, send, 100, iMax-100, 100, -1, 0, true, -1, 99)
		assertPoolExchange(t, send, 100, iMax-99, 100, -1, 0, false, -1, -1)

		//   high reserves but low deposit
		assertPoolExchange(t, send, iMax-100, 100, iMax-100, -1, 0, true, -1, 99)
		assertPoolExchange(t, send, iMax-99, 100, iMax-100, -1, 0, false, -1, -1)

		// fromPool = 0
		assertPoolExchange(t, send, 100, 2, 100, -1, 0, true, -1, 1)
		assertPoolExchange(t, send, 100, 1, 100, -1, 0, false, -1, -1)
	})

	t.Run("disburse edge cases", func(t *testing.T) {
		// Receiving maxReceiveFromPool would deplete the reserves entirely:
		//   low reserves and low maxReceive
		assertPoolExchange(t, recv, 100, -1, 100, 99, 0, true, 9900, -1)
		assertPoolExchange(t, recv, 100, -1, 100, 100, 0, false, -1, -1)

		//   high reserves and high maxReceive
		assertPoolExchange(t, recv, 100, -1, iMax/100, iMax/100-1, 0, true, iMax-107, -1)
		assertPoolExchange(t, recv, 100, -1, iMax/100, iMax/100, 0, false, -1, -1)

		// If
		//           fromPool = k*(maxBps - feeBps) and
		//   reservesFromPool = fromPool + 1
		// then
		//     (reservesToPool * fromPool) / (maxBps - feeBps)
		//         = k * reservesToPool
		// so if k = 101 and reservesToPool = iMax / 100 then B / C > iMax during division
		assertPoolExchange(t, recv, iMax/100, -1, 101*10000+1, 101*10000, 0, false, -1, -1)

		// If
		//            fromPool = maxBps - feeBps and
		//    reservesFromPool = fromPool + 1
		// then
		//      toPool = maxBps * reservesToPool
		// so if reservesToPool = iMax / 100 then the division overflows.
		assertPoolExchange(t, recv, iMax/100, -1, 10000+1, 10000, 0, false, -1, -1)

		// Pool receives more than it has available reserves for
		assertPoolExchange(t, recv, iMax-100, -1, iMax/2, 49, 0, true, 98, -1)
		assertPoolExchange(t, recv, iMax-100, -1, iMax/2, 50, 0, false, -1, -1)
	})

	t.Run("No fees", func(t *testing.T) {
		// Deposits
		assertPoolExchange(t, send, 100, 100, 100, -1, 0, true, -1, 50)      // work exactly
		assertPoolExchange(t, send, 100, 50, 100, -1, 0, true, -1, 33)       // require sending
		assertPoolExchange(t, send, 100, 0, 100, -1, 0, false, -1, -1)       // sending 0
		assertPoolExchange(t, send, 100, iMax-99, 100, -1, 0, false, -1, -1) // sending too much

		// Disburses
		assertPoolExchange(t, recv, 100, -1, 100, 50, 0, true, 100, -1)  // work exactly
		assertPoolExchange(t, recv, 100, -1, 100, 33, 0, true, 50, -1)   // require recving
		assertPoolExchange(t, recv, 100, -1, 100, 0, 0, true, 0, -1)     // receiving 0
		assertPoolExchange(t, recv, 100, -1, 100, 100, 0, false, -1, -1) // receiving too much
	})

	// These test cases look weird because they actually charge 31 bps instead
	// of 30 bps. But this is expected, because you pay fees on the fees you
	// provided: I want to send 10000 after fees, so I send 100030.... but that
	// doesn't work because 0.997 * 10030 = 9999.910 is too low.
	t.Run("30 bps fee actually charges 30 bps", func(t *testing.T) {

		// With no fee, sending 10000 would receive 10000. So to receive
		// 1000 we need to send ceil(10000 / 0.997) = 10031.
		assertPoolExchange(t, send, 10000, 10031, 20000, -1, 30, true, -1, 10000)

		// With no fee, sending 10000 would receive 10000. So to send
		// ceil(10000 / 0.997) = 10031 we need to receive 10000.
		assertPoolExchange(t, recv, 10000, -1, 20000, 10000, 30, true, 10031, -1)
	})
}

// assertPoolExchange validates that pool inputs match their expected outputs.
func assertPoolExchange(t *testing.T,
	exchangeType int,
	reservesBeingDeposited, deposited xdr.Int64,
	reservesBeingDisbursed, disbursed xdr.Int64,
	poolFeeBips xdr.Int32,
	expectedReturn bool, expectedDeposited, expectedDisbursed xdr.Int64,
) {
	var ok bool
	toPool, fromPool := xdr.Int64(-1), xdr.Int64(-1)

	switch exchangeType {
	case tradeTypeDeposit:
		fromPool, ok = calculatePoolPayout(
			reservesBeingDeposited, reservesBeingDisbursed,
			deposited, poolFeeBips)

	case tradeTypeExpectation:
		toPool, ok = calculatePoolExpectation(
			reservesBeingDeposited, reservesBeingDisbursed,
			disbursed, poolFeeBips)

	default:
		t.FailNow()
	}

	if expectedReturn && assert.Equal(t, expectedReturn, ok, "wrong exchange success state") {
		assert.EqualValues(t, expectedDisbursed, fromPool, "wrong payout")
		assert.EqualValues(t, expectedDeposited, toPool, "wrong expectation")
	}
}
