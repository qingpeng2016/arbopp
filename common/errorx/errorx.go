package errorx

// RespErr 自定义响应错误
type RespErr struct {
	Code   int64       `json:"code"`   // 错误码
	Msg    string      `json:"msg"`    // 中文错误信息
	EnMsg  string      `json:"en_msg"` // 英文错误信息
	Data   interface{} `json:"data,omitempty"`
}

func (e *RespErr) Error() string {
	return e.Msg
}

func NewRespErr(code int64, msg, enMsg string) *RespErr {
	return &RespErr{
		Code:  code,
		Msg:   msg,
		EnMsg: enMsg,
	}
}
