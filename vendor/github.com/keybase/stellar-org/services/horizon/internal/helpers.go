package horizon

import (
	"github.com/stellar/go/support/errors"
)

func countNonEmpty(params ...interface{}) (int, error) {
	count := 0

	for _, param := range params {
		switch param := param.(type) {
		default:
			return 0, errors.Errorf("unexpected type %T", param)
		case int32:
			if param != int32(0) {
				count++
			}
		case int64:
			if param != int64(0) {
				count++
			}
		case string:
			if param != "" {
				count++
			}
		}
	}

	return count, nil
}
