package predict

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func orderTypedData(order map[string]any, negRisk, yield bool) apitypes.TypedData {
	contract := exchangeContract(negRisk, yield)
	types := apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"Order": {
			{Name: "salt", Type: "uint256"},
			{Name: "maker", Type: "address"},
			{Name: "signer", Type: "address"},
			{Name: "taker", Type: "address"},
			{Name: "tokenId", Type: "uint256"},
			{Name: "makerAmount", Type: "uint256"},
			{Name: "takerAmount", Type: "uint256"},
			{Name: "expiration", Type: "uint256"},
			{Name: "nonce", Type: "uint256"},
			{Name: "feeRateBps", Type: "uint256"},
			{Name: "side", Type: "uint8"},
			{Name: "signatureType", Type: "uint8"},
		},
	}
	return apitypes.TypedData{
		Types:       types,
		PrimaryType: "Order",
		Domain: apitypes.TypedDataDomain{
			Name:              protocolName,
			Version:           protocolVersion,
			ChainId:           math.NewHexOrDecimal256(chainIDBnbMainnet),
			VerifyingContract: contract,
		},
		Message: order,
	}
}

func orderHash(order map[string]any, negRisk, yield bool) (common.Hash, error) {
	td := orderTypedData(order, negRisk, yield)
	h, _, err := apitypes.TypedDataAndHash(td)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(h), nil
}

func hashKernelMessage(messageHash common.Hash) common.Hash {
	typeHash := crypto.Keccak256([]byte("Kernel(bytes32 hash)"))
	enc := make([]byte, 64)
	copy(enc[0:32], typeHash)
	copy(enc[32:64], messageHash.Bytes())
	return common.BytesToHash(crypto.Keccak256(enc))
}

func eip712WrapHash(messageHash common.Hash, predictAccount string) (common.Hash, error) {
	domain := apitypes.TypedDataDomain{
		Name:              "Kernel",
		Version:           "0.3.1",
		ChainId:           math.NewHexOrDecimal256(chainIDBnbMainnet),
		VerifyingContract: predictAccount,
	}
	td := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
		},
		Domain:      domain,
		PrimaryType: "EIP712Domain",
	}
	sep, err := td.HashStruct("EIP712Domain", domain.Map())
	if err != nil {
		return common.Hash{}, err
	}
	inner := hashKernelMessage(messageHash)
	raw := append([]byte{0x19, 0x01}, sep...)
	raw = append(raw, inner.Bytes()...)
	return common.BytesToHash(crypto.Keccak256(raw)), nil
}

func signPredictAccountOrder(orderHash common.Hash, predictAccount string, key *ecdsa.PrivateKey) (string, error) {
	digest, err := eip712WrapHash(orderHash, predictAccount)
	if err != nil {
		return "", err
	}
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	msg := append(prefix, digest.Bytes()...)
	toSign := crypto.Keccak256(msg)
	sig, err := crypto.Sign(toSign, key)
	if err != nil {
		return "", err
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	return "0x01" + strings.TrimPrefix(strings.ToLower(ecdsaValidator), "0x") + hex.EncodeToString(sig), nil
}

func signEOAOrder(order map[string]any, negRisk, yield bool, key *ecdsa.PrivateKey) (string, common.Hash, error) {
	td := orderTypedData(order, negRisk, yield)
	h, _, err := apitypes.TypedDataAndHash(td)
	if err != nil {
		return "", common.Hash{}, err
	}
	sig, err := crypto.Sign(h, key)
	if err != nil {
		return "", common.Hash{}, err
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	return "0x" + hex.EncodeToString(sig), common.BytesToHash(h), nil
}

func signAuthMessageEOA(message string, key *ecdsa.PrivateKey) (string, error) {
	prefix := accounts.TextHash([]byte(message))
	sig, err := crypto.Sign(prefix, key)
	if err != nil {
		return "", err
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	return "0x" + hex.EncodeToString(sig), nil
}

func signAuthMessagePredictAccount(message string, predictAccount string, key *ecdsa.PrivateKey) (string, error) {
	msgHash := common.BytesToHash(accounts.TextHash([]byte(message)))
	digest, err := eip712WrapHash(msgHash, predictAccount)
	if err != nil {
		return "", err
	}
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	body := append(prefix, digest.Bytes()...)
	toSign := crypto.Keccak256(body)
	sig, err := crypto.Sign(toSign, key)
	if err != nil {
		return "", err
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	return "0x01" + strings.TrimPrefix(strings.ToLower(ecdsaValidator), "0x") + hex.EncodeToString(sig), nil
}

func parsePrivateKey(hexKey string) (*ecdsa.PrivateKey, common.Address, error) {
	hexKey = strings.TrimSpace(hexKey)
	hexKey = strings.TrimPrefix(hexKey, "0x")
	if hexKey == "" {
		return nil, common.Address{}, fmt.Errorf("empty private key")
	}
	raw, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, common.Address{}, err
	}
	key, err := crypto.ToECDSA(raw)
	if err != nil {
		return nil, common.Address{}, err
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return key, addr, nil
}

func randomSalt() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	n := int64(b[0])<<24 | int64(b[1])<<16 | int64(b[2])<<8 | int64(b[3])
	if n < 0 {
		n = -n
	}
	n %= maxSalt
	if n == 0 {
		n = 1
	}
	return fmt.Sprintf("%d", n), nil
}

func buildSignedMarketOrder(
	maker, tokenID, feeRateBps string,
	amounts marketAmounts,
	side int64,
	predictAccount string,
	negRisk, yield bool,
	key *ecdsa.PrivateKey,
) (map[string]any, string, int64, error) {
	salt, err := randomSalt()
	if err != nil {
		return nil, "", 0, err
	}
	expiration := time.Now().Unix() + fiveMinutesSec
	msg := map[string]any{
		"salt":          salt,
		"maker":         strings.ToLower(maker),
		"signer":        strings.ToLower(maker),
		"taker":         zeroAddress,
		"tokenId":       tokenID,
		"makerAmount":   amounts.makerAmount.String(),
		"takerAmount":   amounts.takerAmount.String(),
		"expiration":    math.NewHexOrDecimal256(expiration),
		"nonce":         "0",
		"feeRateBps":    feeRateBps,
		"side":          math.NewHexOrDecimal256(side),
		"signatureType": math.NewHexOrDecimal256(0),
	}
	hash, err := orderHash(msg, negRisk, yield)
	if err != nil {
		return nil, "", 0, err
	}
	var sig string
	if predictAccount != "" {
		sig, err = signPredictAccountOrder(hash, predictAccount, key)
	} else {
		sig, _, err = signEOAOrder(msg, negRisk, yield, key)
	}
	if err != nil {
		return nil, "", 0, err
	}
	msg["signature"] = sig
	msg["hash"] = hash.Hex()
	return msg, amounts.pricePerShare.String(), expiration, nil
}