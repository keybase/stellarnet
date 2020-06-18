package horizon

import (
	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/support/render/hal"
)

// Interface verifications
var _ actions.JSONer = (*DataShowAction)(nil)
var _ actions.RawDataResponder = (*DataShowAction)(nil)
var _ actions.EventStreamer = (*DataShowAction)(nil)

// DataShowAction renders a account summary found by its address.
type DataShowAction struct {
	Action
	Address string
	Key     string
	Data    history.Data
}

// JSON is a method for actions.JSON
func (action *DataShowAction) JSON() error {
	action.Do(
		action.loadParams,
		action.loadRecord,
		func() {
			hal.Render(action.W, map[string]string{"value": action.Data.Value.Base64()})
		},
	)
	return action.Err
}

// Raw is a method for actions.Raw
func (action *DataShowAction) Raw() error {
	action.Do(
		action.loadParams,
		action.loadRecord,
		func() {
			action.W.Write(action.Data.Value)
		},
	)
	return action.Err
}

// SSE is a method for actions.SSE
func (action *DataShowAction) SSE(stream *sse.Stream) error {
	action.Do(
		action.loadParams,
		action.loadRecord,
		func() {
			stream.Send(sse.Event{Data: action.Data.Value.Base64()})
		},
	)
	return action.Err
}

func (action *DataShowAction) loadParams() {
	action.Address = action.GetAddress("account_id", actions.RequiredParam)
	action.Key = action.GetString("key")
}

func (action *DataShowAction) loadRecord() {
	data, err := action.HistoryQ().GetAccountDataByName(action.Address, action.Key)
	action.Err = err
	action.Data = data
}
