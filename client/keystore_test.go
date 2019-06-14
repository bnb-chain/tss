package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/binance-chain/tss-lib/keygen"

	"github.com/binance-chain/tss/common"
)

func TestSaveAndLoad(t *testing.T) {
	var wPriv bytes.Buffer
	var wPub bytes.Buffer
	passphrase := "1234qwerasdf"

	var expectedMsg keygen.LocalPartySaveData
	localSaveDataBytes, err := ioutil.ReadFile("example.json")
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(localSaveDataBytes, &expectedMsg)
	if err != nil {
		t.Fatal(err)
	}
	expectedNodeKey, err := ioutil.ReadFile("node_key")
	if err != nil {
		t.Fatal(err)
	}

	Save(&expectedMsg, expectedNodeKey, common.DefaultKDFConfig(), passphrase, &wPriv, &wPub)
	result, nodeKey := Load(passphrase, &wPriv, &wPub)

	if !reflect.DeepEqual(*result, expectedMsg) {
		t.Fatal("local saved data is not expected")
	}

	if !bytes.Equal(nodeKey, expectedNodeKey) {
		t.Fatal("local saved node key is not expected")
	}
}

func TestSaveAndLoadNeg(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("The code did not panic")
		} else if err := r.(error); err.Error() != "could not decrypt key with given passphrase" {
			t.Errorf("The error is not expected")
		}
	}()

	var wPriv bytes.Buffer
	var wPub bytes.Buffer
	passphrase := "1234qwerasdf"

	var expectedMsg keygen.LocalPartySaveData
	bytes, err := ioutil.ReadFile("example.json")
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(bytes, &expectedMsg)
	if err != nil {
		t.Fatal(err)
	}
	bytes, err = ioutil.ReadFile("node_key")
	if err != nil {
		t.Fatal(err)
	}

	Save(&expectedMsg, bytes, common.DefaultKDFConfig(), passphrase, &wPriv, &wPub)
	Load("12345678", &wPriv, &wPub) // load saved data with a wrong passphrase would not success
}
