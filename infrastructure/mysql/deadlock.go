package mysql

import (
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

const (
	deadlockMaxAttempts = 5
	deadlockBaseBackoff = 50 * time.Millisecond
)

func isDeadlock(err error) bool {
	if err == nil {
		return false
	}
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1213
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "deadlock")
}

func withDeadlockRetry(op func() error) error {
	var err error
	for attempt := 0; attempt < deadlockMaxAttempts; attempt++ {
		err = op()
		if err == nil {
			return nil
		}
		if !isDeadlock(err) || attempt == deadlockMaxAttempts-1 {
			return err
		}
		time.Sleep(deadlockBaseBackoff * time.Duration(1<<attempt))
	}
	return err
}
