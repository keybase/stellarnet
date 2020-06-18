package actions

import (
	"net/http"

	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

func HistoryQFromRequest(request *http.Request) (*history.Q, error) {
	ctx := request.Context()
	session, ok := ctx.Value(&horizonContext.SessionContextKey).(*db.Session)
	if !ok {
		return nil, errors.New("missing session in request context")
	}
	return &history.Q{session}, nil
}
