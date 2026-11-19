package polymarket

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

// TestOrderEIP712Domain 守住下单签名的 EIP-712 domain 字段。
// 背景：CLOB V2（pUSD 抵押 + 0xE111… 交易所）要求 domain version="2"。
// 曾把这里硬编码成 "1"，导致所有下单被交易所判为 invalid POLY_PROXY signature，
// 所以字段和 domain separator 都锁死，改动必须显式更新本用例。
func TestOrderEIP712Domain(t *testing.T) {
	cases := []struct {
		name              string
		negRisk           bool
		verifyingContract string
	}{
		{"standard", false, ctfExchangeAddress},
		{"negRisk", true, negRiskCTFExchangeAddress},
	}
	for _, tc := range cases {
		domain := orderEIP712Domain(tc.negRisk)
		if got := domain["name"]; got != eip712DomainName {
			t.Errorf("%s: name=%v, want %s", tc.name, got, eip712DomainName)
		}
		if got := domain["version"]; got != "2" {
			t.Errorf("%s: version=%v, want 2（V1 会被 CLOB 判为 invalid signature）", tc.name, got)
		}
		if got := domain["chainId"]; got != eip712ChainID {
			t.Errorf("%s: chainId=%v, want %d", tc.name, got, eip712ChainID)
		}
		if got := domain["verifyingContract"]; got != tc.verifyingContract {
			t.Errorf("%s: verifyingContract=%v, want %s", tc.name, got, tc.verifyingContract)
		}
	}
}

// TestOrderEIP712DomainSeparator 锁定 domain separator 的具体取值：
// 只要 name / version / chainId / verifyingContract 任一变化，哈希就会变，用例即失败。
// 期望值取自 CLOB V2（version=2）的实际签名域，改动需同步更新本用例。
func TestOrderEIP712DomainSeparator(t *testing.T) {
	p := &Polymarket{}
	types := orderEIP712DomainTypes()

	want := map[bool]string{
		false: "f549b17a971b3101ede21161e6a67df85b903632bbeec0e5be00b87e982c57a1", // ctfExchangeAddress
		true:  "a6f245fdfe7aaec5c42ef3ab51da02637b29a59933787a695000cd32050dbb4f", // negRiskCTFExchangeAddress
	}
	for _, negRisk := range []bool{false, true} {
		encoded, err := p.encodeData("EIP712Domain", orderEIP712Domain(negRisk), types)
		if err != nil {
			t.Fatalf("encode domain (negRisk=%v): %v", negRisk, err)
		}
		got := hex.EncodeToString(crypto.Keccak256(encoded))
		if got != want[negRisk] {
			t.Errorf("domain separator (negRisk=%v)=%s, want %s", negRisk, got, want[negRisk])
		}
	}
}

// TestOrderEIP712DomainV1Differs 记录这次 bug 的形态：
// 同样字段下 version 变成 "1" 会得到完全不同的 separator，签出的订单交易所不认。
func TestOrderEIP712DomainV1Differs(t *testing.T) {
	p := &Polymarket{}
	types := orderEIP712DomainTypes()

	v2 := orderEIP712Domain(false)
	v1 := orderEIP712Domain(false)
	v1["version"] = "1"

	encodedV2, err := p.encodeData("EIP712Domain", v2, types)
	if err != nil {
		t.Fatalf("encode v2: %v", err)
	}
	encodedV1, err := p.encodeData("EIP712Domain", v1, types)
	if err != nil {
		t.Fatalf("encode v1: %v", err)
	}
	if crypto.Keccak256Hash(encodedV1) == crypto.Keccak256Hash(encodedV2) {
		t.Fatal("version 1 与 2 的 domain separator 不应相同")
	}
}

// TestDomainTypeStringMatchesHelper 保证 poly1271Digest 手写的 domainTypeStringV2
// 与 orderEIP712DomainTypes() 的字段顺序/类型完全一致，避免两处定义漂移。
func TestDomainTypeStringMatchesHelper(t *testing.T) {
	fields := orderEIP712DomainTypes()["EIP712Domain"]

	raw := strings.TrimSuffix(strings.TrimPrefix(domainTypeStringV2, "EIP712Domain("), ")")
	parts := strings.Split(raw, ",")
	if len(parts) != len(fields) {
		t.Fatalf("domainTypeStringV2 字段数=%d, helper=%d", len(parts), len(fields))
	}
	for i, part := range parts {
		kv := strings.Fields(part) // "string name" -> ["string","name"]
		if len(kv) != 2 {
			t.Fatalf("第 %d 项格式异常: %q", i, part)
		}
		if kv[0] != fields[i]["type"] || kv[1] != fields[i]["name"] {
			t.Errorf("第 %d 项: domainTypeStringV2=%s %s, helper=%s %s",
				i, kv[0], kv[1], fields[i]["type"], fields[i]["name"])
		}
	}
}

// TestL1AuthDomainIsSeparate 记录两套 domain 的区别：L1 认证用 ClobAuthDomain（无 verifyingContract），
// 下单用 Polymarket CTF Exchange，二者不可混用。
func TestL1AuthDomainIsSeparate(t *testing.T) {
	l1 := l1AuthDomain()
	if l1["name"] != "ClobAuthDomain" {
		t.Errorf("L1 domain name=%v, want ClobAuthDomain", l1["name"])
	}
	if l1["version"] != clobAuthDomainVersion || clobAuthDomainVersion != "1" {
		t.Errorf("L1 domain version=%v, want 1", l1["version"])
	}
	if _, ok := l1["verifyingContract"]; ok {
		t.Error("L1 auth domain 不应包含 verifyingContract")
	}

	order := orderEIP712Domain(false)
	if order["name"] == l1["name"] {
		t.Error("下单 domain 与 L1 auth domain 不应同名")
	}
	if _, ok := order["verifyingContract"]; !ok {
		t.Error("下单 domain 必须包含 verifyingContract")
	}
}
