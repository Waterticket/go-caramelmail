package caramelmail

import "fmt"

type MailError struct {
	StatusCode int

	Err error
}

func (r *MailError) Error() string {
	return fmt.Sprintf("status %d: err %v", r.StatusCode, r.Err)
}
