package client

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"path"

	"github.com/binance-chain/tss-lib/crypto"
	"github.com/binance-chain/tss-lib/crypto/paillier"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/btcsuite/btcd/btcec"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/sha3"

	tmCrypto "github.com/tendermint/tendermint/crypto"

	"github.com/binance-chain/tss/common"
)

const (
	cipherAlg = "aes-256-ctr"

	// This is essentially a hybrid of the Argon2d and Argon2i algorithms and uses a combination of
	// data-independent memory access (for resistance against side-channel timing attacks) and
	// data-depending memory access (for resistance against GPU cracking attacks).
	keyHeaderKDF = "Argon2id"
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
	Xi         *big.Int             // xi, kj
	PaillierSk *paillier.PrivateKey // ski
	NodeKey    []byte
}

// derived from keygen.LocalPartySaveData
type publicFields struct {
	ShareID           *big.Int
	BigXj             []*crypto.ECPoint     // Xj
	ECDSAPub          *crypto.ECPoint       // y
	PaillierPks       []*paillier.PublicKey // pkj
	NTildej, H1j, H2j []*big.Int
	Index             int
	Ks                []*big.Int
}

// crypto.ECPoint is not json marshallable
func (data *publicFields) MarshalJSON() ([]byte, error) {
	bigXj, err := crypto.FlattenECPoints(data.BigXj)
	if err != nil {
		return nil, errors.New("failed to flatten bigXjs")
	}
	ecdsaPub, err := crypto.FlattenECPoints([]*crypto.ECPoint{data.ECDSAPub})
	if err != nil {
		return nil, errors.New("failed to flatten ecdsa public key")
	}

	type Alias publicFields
	return json.Marshal(&struct {
		BigXj    []*big.Int
		ECDSAPub []*big.Int
		*Alias
	}{
		BigXj:    bigXj,
		ECDSAPub: ecdsaPub,
		Alias:    (*Alias)(data),
	})
}

func (data *publicFields) UnmarshalJSON(payload []byte) error {
	type Alias publicFields
	aux := &struct {
		BigXj    []*big.Int
		ECDSAPub []*big.Int
		*Alias
	}{
		Alias: (*Alias)(data),
	}
	if err := json.Unmarshal(payload, &aux); err != nil {
		return err
	}
	if bigXj, err := crypto.UnFlattenECPoints(tss.EC(), aux.BigXj); err == nil {
		data.BigXj = bigXj
	} else {
		return err
	}
	if pub, err := crypto.UnFlattenECPoints(tss.EC(), aux.ECDSAPub); err == nil && len(pub) == 1 {
		data.ECDSAPub = pub[0]
	} else {
		return err
	}
	return nil
}

// Split LocalPartySaveData into priv.json and pub.json
// where priv.json is
func Save(keygenResult *keygen.LocalPartySaveData, nodeKey []byte, config common.KDFConfig, passphrase string, wPriv, wPub io.Writer) {
	sFields := secretFields{
		keygenResult.Xi,
		keygenResult.PaillierSk,
		nodeKey,
	}

	priv, err := json.Marshal(sFields)
	if err != nil {
		panic(err)
	}

	encrypted, err := encryptSecret(priv, []byte(passphrase), config)
	if err != nil {
		panic(err)
	}
	_, err = wPriv.Write(encrypted)
	if err != nil {
		panic(err)
	}

	pFields := publicFields{
		keygenResult.ShareID,
		keygenResult.BigXj,
		keygenResult.ECDSAPub,
		keygenResult.PaillierPks,
		keygenResult.NTildej,
		keygenResult.H1j,
		keygenResult.H2j,
		keygenResult.Index,
		keygenResult.Ks,
	}

	pub, err := json.Marshal(&pFields)
	if err != nil {
		panic(err)
	}
	_, err = wPub.Write(pub)
	if err != nil {
		panic(err)
	}
}

func Load(passphrase string, rPriv, rPub io.Reader) (saveData *keygen.LocalPartySaveData, nodeKey []byte) {
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
		pFields.ShareID,
		sFields.PaillierSk,

		pFields.BigXj,
		pFields.ECDSAPub,
		pFields.PaillierPks,

		pFields.NTildej,
		pFields.H1j,
		pFields.H2j,
		pFields.Index,
		pFields.Ks,
	}, sFields.NodeKey
}

func LoadPubkey(home string) (tmCrypto.PubKey, error) {
	var pFields publicFields
	pBytes, err := ioutil.ReadFile(path.Join(home, "pk.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(pBytes, &pFields); err != nil {
		return nil, err
	}
	ecdsaPubKey := &ecdsa.PublicKey{tss.EC(), pFields.ECDSAPub.X(), pFields.ECDSAPub.Y()}
	btcecPubKey := (*btcec.PublicKey)(ecdsaPubKey)

	var pubkeyBytes secp256k1.PubKeySecp256k1
	copy(pubkeyBytes[:], btcecPubKey.SerializeCompressed())
	return pubkeyBytes, nil
}

func encryptSecret(data, auth []byte, config common.KDFConfig) ([]byte, error) {
	salt := make([]byte, config.SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	derivedKey := argon2.IDKey(auth, salt, config.Iterations, config.Memory, config.Parallelism, config.KeyLength)
	encryptKey := derivedKey[:len(derivedKey)-16]

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	cipherText, err := aesCTRXOR(encryptKey, data, iv)
	if err != nil {
		return nil, err
	}

	d := sha3.New256()
	d.Write(derivedKey[len(derivedKey)-16:])
	d.Write(cipherText)
	mac := d.Sum(nil)

	argon2ParamsJSON := make(map[string]interface{}, 5)
	argon2ParamsJSON["i"] = config.Iterations
	argon2ParamsJSON["m"] = config.Memory
	argon2ParamsJSON["p"] = config.Parallelism
	argon2ParamsJSON["dklen"] = config.KeyLength
	argon2ParamsJSON["salt"] = hex.EncodeToString(salt)

	cipherParamsJSON := cipherparamsJSON{
		IV: hex.EncodeToString(iv),
	}

	cryptoStruct := cryptoJSON{
		Cipher:       cipherAlg,
		CipherText:   hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON,
		KDF:          keyHeaderKDF,
		KDFParams:    argon2ParamsJSON,
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
	d.Write(derivedKey[len(derivedKey)-16:])
	d.Write(cipherText)
	calculatedMAC := d.Sum(nil)

	if !bytes.Equal(calculatedMAC, mac) {
		return nil, errors.New("could not decrypt key with given passphrase")
	}

	plainText, err := aesCTRXOR(derivedKey[:len(derivedKey)-16], cipherText, iv)
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
	dkLen := ensureUInt32(encryptedSecret.KDFParams["dklen"])
	i := ensureUInt32(encryptedSecret.KDFParams["i"])
	m := ensureUInt32(encryptedSecret.KDFParams["m"])
	p := ensureUInt8(encryptedSecret.KDFParams["p"])
	return argon2.IDKey(authArray, salt, i, m, p, dkLen), nil
}

func ensureUInt32(x interface{}) uint32 {
	res, ok := x.(uint32)
	if !ok {
		res = uint32(x.(float64))
	}
	return res
}

func ensureUInt8(x interface{}) uint8 {
	res, ok := x.(uint8)
	if !ok {
		res = uint8(x.(float64))
	}
	return res
}
