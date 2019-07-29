package cmd

import (
	"testing"
)

func TestTimestampHexConvertion(t *testing.T) {
	expected := 1564372501
	encoded := convertTimestampToHex(int64(expected))
	if encoded != "5D3E6E15" {
		t.Fail()
	}
	timestamp := convertHexToTimestamp(encoded)
	if timestamp != expected {
		t.Fail()
	}
}

func TestMultiAddrStrToNormalAddr(t *testing.T) {
	res, err := convertMultiAddrStrToNormalAddr("/ip4/127.0.0.1/tcp/27148")
	if err != nil {
		t.Fail()
	}
	if res != "127.0.0.1:27148" {
		t.Fail()
	}
}
