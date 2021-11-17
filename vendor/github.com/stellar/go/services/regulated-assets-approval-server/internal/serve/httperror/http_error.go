package httperror

import (
	"net/http"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/httpjson"
)

type Error struct {
	ErrorMessage string `json:"error"`
	Status       int    `json:"-"`
}

func (h *Error) Error() string {
	return h.ErrorMessage
}

func NewHTTPError(status int, errorMessage string) *Error {
	return &Error{
		ErrorMessage: errorMessage,
		Status:       status,
	}
}

func (e *Error) Render(w http.ResponseWriter) {
	httpjson.RenderStatus(w, e.Status, e, httpjson.JSON)
}

var InternalServer = &Error{
	ErrorMessage: "An error occurred while processing this request.",
	Status:       http.StatusInternalServerError,
}

var BadRequest = &Error{
	ErrorMessage: "The request was invalid in some way.",
	Status:       http.StatusBadRequest,
}

func ParseHorizonError(err error) error {
	if err == nil {
		return nil
	}

	rootErr := errors.Cause(err)
	if hError := horizonclient.GetError(rootErr); hError != nil {
		resultCode, _ := hError.ResultCodes()
		err = errors.Wrapf(err, "error submitting transaction: %+v, %+v\n", hError.Problem, resultCode)
	} else {
		err = errors.Wrap(err, "error submitting transaction")
	}
	return err
}
