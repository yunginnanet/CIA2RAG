package errs

import (
	"errors"
	"fmt"
)

type BadStatusCodeError struct {
	StatusCode int
}

func (b BadStatusCodeError) Error() string {
	return fmt.Sprintf("bad status code: %d", b.StatusCode)
}

func AsBadStatusCodeError(err *error) (int, bool) {
	bse := BadStatusCodeError{}
	if errors.As(*err, &bse) {
		return bse.StatusCode, true
	}
	return 0, false
}
