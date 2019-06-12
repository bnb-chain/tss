package client

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"

	"github.com/binance-chain/tss-lib/crypto/paillier"
	"github.com/binance-chain/tss-lib/keygen"
	"github.com/binance-chain/tss-lib/types"
	"golang.org/x/crypto/scrypt"
	"golang.org/x/crypto/sha3"
)

const (
	cipherAlg = "aes-256-ctr"

	keyHeaderKDF = "scrypt"

	// scryptN is the N parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	scryptN = 1 << 18

	// scryptP is the P parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	scryptP = 1

	scryptR     = 8
	scryptDKLen = 48
)

type cryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

// derived from keygen.LocalPartySaveData
type secretFields struct {
	Xi, ShareID *big.Int             // xi, kj
	PaillierSk  *paillier.PrivateKey // ski
}

// derived from keygen.LocalPartySaveData
type publicFields struct {
	BigXj             []*types.ECPoint      // Xj
	ECDSAPub          *types.ECPoint        // y
	PaillierPks       []*paillier.PublicKey // pkj
	NTildej, H1j, H2j []*big.Int
}

// Split LocalPartySaveData into priv.json and pub.json
// where priv.json is
func Save(keygenResult *keygen.LocalPartySaveData, passphrase string, wPriv, wPub io.Writer) {
	sFields := secretFields{
		keygenResult.Xi,
		keygenResult.ShareID,
		keygenResult.PaillierSk,
	}

	priv, err := json.Marshal(sFields)
	if err != nil {
		panic(err)
	}

	encrypted, err := encryptSecret(priv, []byte(passphrase))
	if err != nil {
		panic(err)
	}
	_, err = wPriv.Write(encrypted)
	if err != nil {
		panic(err)
	}

	pFields := publicFields{
		keygenResult.BigXj,
		keygenResult.ECDSAPub,
		keygenResult.PaillierPks,
		keygenResult.NTildej,
		keygenResult.H1j,
		keygenResult.H2j,
	}

	pub, err := json.Marshal(pFields)
	if err != nil {
		panic(err)
	}
	_, err = wPub.Write(pub)
	if err != nil {
		panic(err)
	}
}

func Load(passphrase string, rPriv, rPub io.Reader) *keygen.LocalPartySaveData {
	var encryptedSecret cryptoJSON
	var pFields publicFields

	sBytes, err := ioutil.ReadAll(rPriv)
	if err != nil {
		panic(fmt.Errorf("failed to load private bytes from file: %v", err))
	}

	err = json.Unmarshal(sBytes, &encryptedSecret)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal secret bytes: %v", err))
	}

	sFields, err := decryptSecret(encryptedSecret, passphrase)
	if err != nil {
		panic(err)
	}

	pBytes, err := ioutil.ReadAll(rPub)
	if err != nil {
		panic(fmt.Errorf("failed to load public bytes from file: %v", err))
	}

	err = json.Unmarshal(pBytes, &pFields)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal public bytes: %v", err))
	}

	return &keygen.LocalPartySaveData{
		sFields.Xi,
		sFields.ShareID,
		sFields.PaillierSk,

		pFields.BigXj,
		pFields.ECDSAPub,
		pFields.PaillierPks,

		pFields.NTildej,
		pFields.H1j,
		pFields.H2j,
	}
}

func encryptSecret(data, auth []byte) ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	derivedKey, err := scrypt.Key(auth, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}
	encryptKey := derivedKey[:32]

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	cipherText, err := aesCTRXOR(encryptKey, data, iv)
	if err != nil {
		return nil, err
	}

	d := sha3.New256()
	d.Write(derivedKey[32:48])
	d.Write(cipherText)
	mac := d.Sum(nil)

	scryptParamsJSON := make(map[string]interface{}, 5)
	scryptParamsJSON["n"] = scryptN
	scryptParamsJSON["r"] = scryptR
	scryptParamsJSON["p"] = scryptP
	scryptParamsJSON["dklen"] = scryptDKLen
	scryptParamsJSON["salt"] = hex.EncodeToString(salt)

	cipherParamsJSON := cipherparamsJSON{
		IV: hex.EncodeToString(iv),
	}

	cryptoStruct := cryptoJSON{
		Cipher:       cipherAlg,
		CipherText:   hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON,
		KDF:          keyHeaderKDF,
		KDFParams:    scryptParamsJSON,
		MAC:          hex.EncodeToString(mac),
	}
	return json.Marshal(cryptoStruct)
}

func decryptSecret(encryptedSecret cryptoJSON, passphrase string) (*secretFields, error) {
	if encryptedSecret.Cipher != cipherAlg {
		return nil, fmt.Errorf("Cipher not supported: %s", encryptedSecret.Cipher)
	}
	mac, err := hex.DecodeString(encryptedSecret.MAC)
	if err != nil {
		return nil, err
	}

	iv, err := hex.DecodeString(encryptedSecret.CipherParams.IV)
	if err != nil {
		return nil, err
	}

	cipherText, err := hex.DecodeString(encryptedSecret.CipherText)
	if err != nil {
		return nil, err
	}

	derivedKey, err := getKDFKey(encryptedSecret, passphrase)
	if err != nil {
		return nil, err
	}

	d := sha3.New256()
	d.Write(derivedKey[32:48])
	d.Write(cipherText)
	calculatedMAC := d.Sum(nil)

	if !bytes.Equal(calculatedMAC, mac) {
		return nil, errors.New("could not decrypt key with given passphrase")
	}

	plainText, err := aesCTRXOR(derivedKey[:32], cipherText, iv)
	if err != nil {
		return nil, err
	}
	var sFields secretFields
	err = json.Unmarshal(plainText, &sFields)
	if err != nil {
		return nil, err
	}

	return &sFields, nil
}

func aesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	// AES-256 is selected due to size of encryptKey.
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, err
}

func getKDFKey(encryptedSecret cryptoJSON, auth string) ([]byte, error) {
	authArray := []byte(auth)
	salt, err := hex.DecodeString(encryptedSecret.KDFParams["salt"].(string))
	if err != nil {
		return nil, err
	}
	dkLen := ensureInt(encryptedSecret.KDFParams["dklen"])

	n := ensureInt(encryptedSecret.KDFParams["n"])
	r := ensureInt(encryptedSecret.KDFParams["r"])
	p := ensureInt(encryptedSecret.KDFParams["p"])
	return scrypt.Key(authArray, salt, n, r, p, dkLen)
}

func ensureInt(x interface{}) int {
	res, ok := x.(int)
	if !ok {
		res = int(x.(float64))
	}
	return res
}
