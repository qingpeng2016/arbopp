package response

import (
	"errors"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/errorx"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ApiResp struct {
	Code    uint32 `json:"code"`
	Data    interface{}
	Message string `json:"message"`
}

const (
	OkBizCode uint32 = 200
)

// ResponseErr 统一错误响应处理
func ResponseErr(c *gin.Context, err error) {
	var bizErr *errorx.RespErr
	if errors.As(err, &bizErr) {
		// 业务错误：使用 RespErr 的错误码和英文消息
		Response(c, http.StatusOK, uint32(bizErr.Code), bizErr.Data, bizErr.EnMsg)
		return
	}

	// 普通错误：使用默认未知错误
	Response(c, http.StatusOK, uint32(errorx.ErrUnknown.Code), nil, errorx.ErrUnknown.EnMsg)
}

// ResponseSuccess 成功响应
func ResponseSuccess(c *gin.Context, data interface{}) {
	Response(c, http.StatusOK, OkBizCode, data, "success")
}

// ResponseSuccessWithMsg 成功响应（自定义消息）
func ResponseSuccessWithMsg(c *gin.Context, data interface{}, msg string) {
	Response(c, http.StatusOK, OkBizCode, data, msg)
}

// Response 统一响应方法
func Response(c *gin.Context, httpCode, errCode uint32, data interface{}, message string) {
	c.JSON(int(httpCode), gin.H{
		"code":    errCode,
		"message": message,
		"data":    data,
	})
	c.Abort()
}
