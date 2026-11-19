package mysqlclient

import "github.com/go-sql-driver/mysql"

// IsDuplicateError is duplicate error
func IsDuplicateError(err error) bool {
	if mysqlErr, ok := err.(*mysql.MySQLError); !ok {
		return false
	} else if mysqlErr.Number == 1062 {
		return true
	}
	return false
}
