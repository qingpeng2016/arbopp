package errorx

var (
	// 通用错误
	ErrParamsError = NewRespErr(100100, "参数错误", "Params error.")
	ErrUnknown     = NewRespErr(100101, "未知错误，请联系客服", "Unknown error. Please contact customer.")
	ErrDbError     = NewRespErr(100109, "数据库错误", "Database error")
)
