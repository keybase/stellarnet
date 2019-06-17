package ticker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/stellar/go/services/ticker/internal/tickerdb"
	"github.com/stellar/go/services/ticker/internal/utils"
	hlog "github.com/stellar/go/support/log"
)

// GenerateMarketSummaryFile generates a MarketSummary with the statistics for all
// valid markets within the database and outputs it to <filename>.
func GenerateMarketSummaryFile(s *tickerdb.TickerSession, l *hlog.Entry, filename string, forCMC bool) error {
	l.Infoln("Generating market data...")
	marketSummary, err := GenerateMarketSummary(s)
	if err != nil {
		return err
	}
	l.Infoln("Market data successfully generated!")

	if forCMC {
		l.Infoln("Inverting CMC markets...")
		invertCMCMarkets(&marketSummary)
		l.Infoln("Markets successfully adapted for CoinMarketCap")
	}

	jsonMkt, err := json.MarshalIndent(marketSummary, "", "    ")
	if err != nil {
		return err
	}

	l.Infoln("Writing market data to: ", filename)
	numBytes, err := utils.WriteJSONToFile(jsonMkt, filename)
	if err != nil {
		return err
	}
	l.Infof("Wrote %d bytes to %s\n", numBytes, filename)
	return nil
}

// GenerateMarketSummary outputs a MarketSummary with the statistics for all
// valid markets within the database.
func GenerateMarketSummary(s *tickerdb.TickerSession) (ms MarketSummary, err error) {
	var marketStatsSlice []MarketStats
	now := time.Now()
	nowMillis := utils.TimeToUnixEpoch(now)
	nowRFC339 := utils.TimeToRFC3339(now)

	dbMarkets, err := s.RetrieveMarketData()
	if err != nil {
		return
	}

	for _, dbMarket := range dbMarkets {
		marketStats := dbMarketToMarketStats(dbMarket)
		marketStatsSlice = append(marketStatsSlice, marketStats)
	}

	ms = MarketSummary{
		GeneratedAt:        nowMillis,
		GeneratedAtRFC3339: nowRFC339,
		Pairs:              marketStatsSlice,
	}
	return
}

func dbMarketToMarketStats(m tickerdb.Market) MarketStats {
	closeTime := utils.TimeToRFC3339(m.LastPriceCloseTime)

	spread, spreadMidPoint := utils.CalcSpread(m.HighestBid, m.LowestAsk)
	return MarketStats{
		TradePairName:    m.TradePair,
		BaseVolume24h:    m.BaseVolume24h,
		CounterVolume24h: m.CounterVolume24h,
		TradeCount24h:    m.TradeCount24h,
		Open24h:          m.OpenPrice24h,
		Low24h:           m.LowestPrice24h,
		High24h:          m.HighestPrice24h,
		Change24h:        m.PriceChange24h,
		BaseVolume7d:     m.BaseVolume7d,
		CounterVolume7d:  m.CounterVolume7d,
		TradeCount7d:     m.TradeCount7d,
		Open7d:           m.OpenPrice7d,
		Low7d:            m.LowestPrice7d,
		High7d:           m.HighestPrice7d,
		Change7d:         m.PriceChange7d,
		Price:            m.LastPrice,
		Close:            m.LastPrice,
		BidCount:         m.NumBids,
		BidVolume:        m.BidVolume,
		BidMax:           m.HighestBid,
		AskCount:         m.NumAsks,
		AskVolume:        m.AskVolume,
		AskMin:           m.LowestAsk,
		Spread:           spread,
		SpreadMidPoint:   spreadMidPoint,
		CloseTime:        closeTime,
	}
}

// This is a set of temporary functions that are used to invert
// some asset pairs when generating the assets.json report to fix
// the prices / volumes on CoinMarketCap.

func invertIfNotNull(n float64) float64 {
	if n != 0.0 {
		return 1.0 / n
	}
	return n
}

func invertMarketStats(m *MarketStats) {
	// Uncomment if we need to change the trade pair order:
	//
	// tradePairItems := strings.Split(m.TradePairName, "_")
	// m.TradePairName = fmt.Sprintf("%s_%s", tradePairItems[1], tradePairItems[0])

	m.Close = invertIfNotNull(m.Close)
	m.Price = m.Close

	// Re-calculating 24h stats:
	m.Open24h = invertIfNotNull(m.Open24h)

	currLow24h := m.Low24h
	currHigh24h := m.High24h

	if currHigh24h != 0.0 {
		m.Low24h = 1.0 / currHigh24h
	}

	if currLow24h != 0.0 {
		m.High24h = 1.0 / currLow24h
	}

	m.Change24h = m.Close - m.Open24h

	// Re-calculating 7d stats:
	m.Open7d = invertIfNotNull(m.Open7d)

	currLow7d := m.Low7d
	currHigh7d := m.High7d

	if currHigh7d != 0.0 {
		m.Low7d = 1.0 / currHigh7d
	}

	if currLow7d != 0.0 {
		m.High7d = 1.0 / currLow7d
	}

	m.Change7d = m.Close - m.Open7d

	// Since we're reversing the asset pairs, the orderbook data is invalidated:
	m.BidCount = 0.0
	m.BidVolume = 0.0
	m.BidMax = 0.0
	m.AskCount = 0.0
	m.AskVolume = 0.0
	m.AskMin = 0.0
	m.Spread = 0.0
	m.SpreadMidPoint = 0.0
}

func shouldMarketBeInvertdForCMC(pairName string) bool {
	cmcMarketPairs := []string{
		"XLM_SHX",
		"XLM_SLT",
		"XLM_DOGET",
		"XLM_RMT",
		"XLM_MOBI",
		"XLM_ETH",
		"XLM_BTC",
		"XLM_KIN",
		"XLM_XRP",
		"XLM_XLB",
		"XLM_TERN",
		"XLM_BTX",
		"XLM_GRAT",
		"XLM_DRA",
		"XLM_PEDI",
		"XLM_ABDT",
		"XLM_ZRX",
		"XLM_WLO",
		"XLM_LTC",
		"XLM_SHADE",
		"XLM_XLMG",
		"XLM_OMG",
		"XLM_ETX",
		"XLM_NEOX",
		"XLM_REPO",
	}
	for _, cmcPair := range cmcMarketPairs {
		if pairName == cmcPair {
			fmt.Println("Found a matching pair")
			return true
		}
	}

	return false
}

func invertCMCMarkets(ms *MarketSummary) {
	for i, mkt := range ms.Pairs {
		if shouldMarketBeInvertdForCMC(mkt.TradePairName) {
			invertMarketStats(&ms.Pairs[i])
		}
	}
}
