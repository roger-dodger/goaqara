package aqara

import (
	"testing"
)

func TestSign(t *testing.T) {
	// Sample credentials from:
	// https://opendoc.aqara.cn/en/docs/developmanual/apiIntroduction/signGenerationRules.html
	accessToken := "532cad73c5493193d63d367016b98b27"
	appID := "4e693d54d75db580a56d1263"
	keyID := "k.78784564654feda454557"
	appKey := "gU7Qtxi4dWnYAdmudyxni52bWZ58b8uN"
	account := "test"
	nonce := "C6wuzd0Qguxzelhb"
	timestamp := "1618914078668"

	expectedSignature := "314a6f6fd46264e6ec872e21f88361c3"

	aqaraClient := New(ServerRegionEurope, appID, appKey, keyID, account, false)
	signature := aqaraClient.sign(accessToken, nonce, timestamp)

	if signature != expectedSignature {
		t.Errorf("got signature %q, wanted signature %q", signature, expectedSignature)
	}
}
