package kalshi

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
)

// signKalshiPSS 对 timestamp+method+path 做 RSA-PSS SHA256，返回标准 base64。
func signKalshiPSS(privateKeyPEM, passphrase, message string) (string, error) {
	key, err := parseRSAPrivateKey(privateKeyPEM, passphrase)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(message))
	sig, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, sum[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		return "", fmt.Errorf("sign kalshi request: %w", err)
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func parseRSAPrivateKey(material, passphrase string) (*rsa.PrivateKey, error) {
	material = strings.TrimSpace(material)
	if material == "" {
		return nil, fmt.Errorf("kalshi api_secret is required")
	}
	var der []byte
	if strings.Contains(material, "BEGIN") {
		block, _ := pem.Decode([]byte(material))
		if block == nil {
			return nil, fmt.Errorf("kalshi api_secret is not a PEM private key")
		}
		der = block.Bytes
		if x509.IsEncryptedPEMBlock(block) {
			plain, err := x509.DecryptPEMBlock(block, []byte(passphrase))
			if err != nil {
				return nil, fmt.Errorf("decrypt kalshi private key: %w", err)
			}
			der = plain
		}
	} else {
		raw, err := base64.StdEncoding.DecodeString(material)
		if err != nil {
			raw, err = base64.RawStdEncoding.DecodeString(material)
		}
		if err != nil {
			return nil, fmt.Errorf("kalshi api_secret is not a PEM or base64 private key")
		}
		der = raw
	}
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("parse kalshi private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("kalshi api_secret is not an RSA private key")
	}
	return key, nil
}
