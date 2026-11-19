package predict

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gph-tech/fgmm-strategy-bitfinex/domain/http/entity"
)

type jwtSlot struct {
	token string
	until time.Time
}

func (c *Client) ensureJWT(ctx context.Context, acc entity.PredictAccount) (string, error) {
	apiKey, priv, addr, predictAcct, err := resolveSigner(acc)
	if err != nil {
		return "", err
	}
	if apiKey == "" {
		apiKey = strings.TrimSpace(c.apiKey)
	}
	cacheKey := strings.ToLower(predictAcct)
	if cacheKey == "" {
		cacheKey = strings.ToLower(addr.Hex())
	}
	c.jwtMu.Lock()
	if slot, ok := c.jwtByKey[cacheKey]; ok && slot.token != "" && time.Now().Before(slot.until) {
		tok := slot.token
		c.jwtMu.Unlock()
		return tok, nil
	}
	c.jwtMu.Unlock()

	var msgResp struct {
		Success bool `json:"success"`
		Data    struct {
			Message string `json:"message"`
		} `json:"data"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := c.doGETWithKey(ctx, apiKey, "/auth/message", nil, &msgResp); err != nil {
		return "", err
	}
	if !msgResp.Success || strings.TrimSpace(msgResp.Data.Message) == "" {
		return "", fmt.Errorf("predict auth message: %s", apiErr(msgResp.Message, msgResp.Error))
	}
	message := msgResp.Data.Message
	signer := strings.ToLower(addr.Hex())
	if predictAcct != "" {
		signer = strings.ToLower(predictAcct)
	}
	var sig string
	if predictAcct != "" {
		sig, err = signAuthMessagePredictAccount(message, predictAcct, priv)
	} else {
		sig, err = signAuthMessageEOA(message, priv)
	}
	if err != nil {
		return "", err
	}
	body := map[string]string{
		"signer":    signer,
		"message":   message,
		"signature": sig,
	}
	var authResp struct {
		Success bool `json:"success"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := c.doPOSTWithKey(ctx, apiKey, "/auth", body, "", &authResp); err != nil {
		return "", err
	}
	if !authResp.Success || strings.TrimSpace(authResp.Data.Token) == "" {
		return "", fmt.Errorf("predict auth: %s", apiErr(authResp.Message, authResp.Error))
	}
	token := strings.TrimSpace(authResp.Data.Token)
	c.jwtMu.Lock()
	if c.jwtByKey == nil {
		c.jwtByKey = map[string]jwtSlot{}
	}
	c.jwtByKey[cacheKey] = jwtSlot{token: token, until: time.Now().Add(23 * time.Hour)}
	c.jwtMu.Unlock()
	return token, nil
}

func resolveSigner(acc entity.PredictAccount) (apiKey string, priv *ecdsa.PrivateKey, addr common.Address, predictAcct string, err error) {
	priv, addr, err = parsePrivateKey(acc.PrivateKey)
	if err != nil {
		return "", nil, common.Address{}, "", err
	}
	if s := strings.TrimSpace(acc.EOAAddress); s != "" {
		addr = common.HexToAddress(s)
	}
	predictAcct = strings.TrimSpace(acc.PredictAccount)
	apiKey = strings.TrimSpace(acc.APIKey)
	return apiKey, priv, addr, predictAcct, nil
}

func apiErr(msg, code string) string {
	msg = strings.TrimSpace(msg)
	code = strings.TrimSpace(code)
	if msg == "" {
		msg = code
	}
	if msg == "" {
		msg = "success=false"
	}
	return msg
}