package pmdata

import (
	"net/http"
	"strings"
)

func IsNotFound(status int) bool {
	return status == http.StatusNotFound
}

func IsForbidden(status int, err error) bool {
	if status == http.StatusForbidden {
		return true
	}
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "http 403")
}