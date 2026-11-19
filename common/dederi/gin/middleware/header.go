package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/gph-tech/fgmm-strategy-bitfinex/common/dederi/languages"
	cryptoPredictionEnum "github.com/gph-tech/fgmm-strategy-bitfinex/common/enum/crypto/prediction"
	"strconv"
)

func GetLanguage(c *gin.Context) languages.Lang {
	return languages.Lang(c.GetHeader("lang"))
}

func GetLang(c *gin.Context) cryptoPredictionEnum.Lang {
	lang, err := strconv.Atoi(c.GetHeader("lang"))
	if err != nil {
		return cryptoPredictionEnum.LangEn
	}

	switch lang {
	case 1:
		return cryptoPredictionEnum.LangEn
	case 2:
		return cryptoPredictionEnum.LangCh
	case 3:
		return cryptoPredictionEnum.LangHk
	default:
		return cryptoPredictionEnum.LangEn
	}
}
