package stellarnet

import (
	"errors"
	"strconv"
)

// FeeStats returns NumericFeeStats given a FeeStatFetcher.
func FeeStats(f FeeStatFetcher) (NumericFeeStats, error) {
	resp, err := f.FeeStatFetch()
	if err != nil {
		return NumericFeeStats{}, err
	}
	return resp.Convert()
}

// FeeStatsResponse describes the json response from the horizon
// /fee_stats endpoint (which is unfortunately all strings).
type FeeStatsResponse struct {
	LastLedger          string              `json:"last_ledger"`
	LastLedgerBaseFee   string              `json:"last_ledger_base_fee"`
	LedgerCapacityUsage string              `json:"ledger_capacity_usage"`
	FeeCharged          FeeStatsSubResponse `json:"fee_charged"`
	MaxFee              FeeStatsSubResponse `json:"max_fee"`
}

type FeeStatsSubResponse struct {
	Min  string `json:"min"`
	Max  string `json:"max"`
	Mode string `json:"mode"`
	P10  string `json:"p10"`
	P20  string `json:"p20"`
	P30  string `json:"p30"`
	P40  string `json:"p40"`
	P50  string `json:"p50"`
	P60  string `json:"p60"`
	P70  string `json:"p70"`
	P80  string `json:"p80"`
	P90  string `json:"p90"`
	P95  string `json:"p95"`
	P99  string `json:"p99"`
}

// Convert converts a FeeStatsResponse into NumericFeeStats by
// converting all the strings to the appropriate numeric types.
func (f FeeStatsResponse) Convert() (x NumericFeeStats, err error) {
	var s NumericFeeStats
	n, err := strconv.ParseInt(f.LastLedger, 10, 32)
	if err != nil {
		return x, err
	}
	s.LastLedger = int32(n)
	s.LastLedgerBaseFee, err = strconv.ParseUint(f.LastLedgerBaseFee, 10, 64)
	if err != nil {
		return x, err
	}
	s.LedgerCapacityUsage, err = strconv.ParseFloat(f.LedgerCapacityUsage, 64)
	if err != nil {
		return x, err
	}
	s.MinAcceptedFee, err = strconv.ParseUint(f.FeeCharged.Min, 10, 64)
	if err != nil {
		return x, err
	}
	s.ModeAcceptedFee, err = strconv.ParseUint(f.FeeCharged.Mode, 10, 64)
	if err != nil {
		return x, err
	}
	s.P10AcceptedFee, err = strconv.ParseUint(f.FeeCharged.P10, 10, 64)
	if err != nil {
		return x, err
	}
	s.P20AcceptedFee, err = strconv.ParseUint(f.FeeCharged.P20, 10, 64)
	if err != nil {
		return x, err
	}
	s.P30AcceptedFee, err = strconv.ParseUint(f.FeeCharged.P30, 10, 64)
	if err != nil {
		return x, err
	}
	s.P40AcceptedFee, err = strconv.ParseUint(f.FeeCharged.P40, 10, 64)
	if err != nil {
		return x, err
	}
	s.P50AcceptedFee, err = strconv.ParseUint(f.FeeCharged.P50, 10, 64)
	if err != nil {
		return x, err
	}
	s.P60AcceptedFee, err = strconv.ParseUint(f.FeeCharged.P60, 10, 64)
	if err != nil {
		return x, err
	}
	s.P70AcceptedFee, err = strconv.ParseUint(f.FeeCharged.P70, 10, 64)
	if err != nil {
		return x, err
	}
	s.P80AcceptedFee, err = strconv.ParseUint(f.FeeCharged.P80, 10, 64)
	if err != nil {
		return x, err
	}
	s.P90AcceptedFee, err = strconv.ParseUint(f.FeeCharged.P90, 10, 64)
	if err != nil {
		return x, err
	}
	s.P95AcceptedFee, err = strconv.ParseUint(f.FeeCharged.P95, 10, 64)
	if err != nil {
		return x, err
	}
	s.P99AcceptedFee, err = strconv.ParseUint(f.FeeCharged.P99, 10, 64)
	if err != nil {
		return x, err
	}

	return s, nil
}

// FeeStatFetcher contains FeeStatFetch, which will get a FeeStatsResponse.
type FeeStatFetcher interface {
	FeeStatFetch() (FeeStatsResponse, error)
}

// NumericFeeStats is a numeric representation of the fee stats.
type NumericFeeStats struct {
	LastLedger          int32
	LastLedgerBaseFee   uint64
	LedgerCapacityUsage float64
	MinAcceptedFee      uint64
	ModeAcceptedFee     uint64
	P10AcceptedFee      uint64
	P20AcceptedFee      uint64
	P30AcceptedFee      uint64
	P40AcceptedFee      uint64
	P50AcceptedFee      uint64
	P60AcceptedFee      uint64
	P70AcceptedFee      uint64
	P80AcceptedFee      uint64
	P90AcceptedFee      uint64
	P95AcceptedFee      uint64
	P99AcceptedFee      uint64
}

// HorizonFeeStatFetcher is a FeeStatFetcher that uses a live horizon
// client.
type HorizonFeeStatFetcher struct{}

// FeeStatFetch implements FeeStatFetcher.
func (h *HorizonFeeStatFetcher) FeeStatFetch() (FeeStatsResponse, error) {
	c := Client()
	if c == nil {
		return FeeStatsResponse{}, errors.New("no horizon client")
	}
	statsURL, err := horizonLink(c.HorizonURL, "/fee_stats")

	var resp FeeStatsResponse
	err = getDecodeJSONStrict(statsURL, c.HTTP.Get, &resp)
	if err != nil {
		return FeeStatsResponse{}, err
	}

	return resp, nil
}
