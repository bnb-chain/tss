package cmd

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bgentry/speakeasy"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/common"
)

type bootstrapper struct {
	channelId     string
	channelPasswd string
	peerAddrs     []string

	peers sync.Map // id -> peerInfo
}

func init() {
	rootCmd.AddCommand(bootstrap)
	bootstrap.AddCommand(channel)
}

var bootstrap = &cobra.Command{
	Use:   "bootstrap",
	Short: "bootstrapping for network configuration",
	Long:  "bootstrapping for network configuration. Will try connect to configured address and get peer's id and moniker",
	PreRun: func(cmd *cobra.Command, args []string) {
		home := viper.GetString("home")
		common.ReadConfigFromHome(viper.GetViper(), home)
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		channelId := askChannelId()
		channelPasswd := askChannelPasswd()
		setN()
		peerAddrs := askPeerAddrs()

		bootstrapper := bootstrapper{
			channelId:     channelId,
			channelPasswd: channelPasswd,
			peerAddrs:     peerAddrs,
		}

		bootstrapMsg, err := NewBootstrapMessage(channelId, channelPasswd, common.TssCfg.Moniker, common.TssCfg.Id, common.TssCfg.ListenAddr)
		if err != nil {
			panic(err)
		}

		src, err := convertMultiAddrStrToNormalAddr(common.TssCfg.ListenAddr)
		if err != nil {
			panic(err)
		}
		listener, _ := net.Listen("tcp", src)

		defer listener.Close()

		done := make(chan bool)

		go func() {
			for {
				conn, err := listener.Accept()
				if err != nil {
					fmt.Printf("Some connection error: %s\n", err)
				} else {
					fmt.Printf("%s connected to us!\n", conn.RemoteAddr().String())
				}

				go bootstrapper.handleConnection(conn, bootstrapMsg)
			}
		}()

		go func() {
			for _, peerAddr := range peerAddrs {
				go func(peerAddr string) {
					dest, err := convertMultiAddrStrToNormalAddr(peerAddr)
					if err != nil {
						panic(fmt.Errorf("failed to convert peer multiAddr to addr: %v", err))
					}
					conn, err := net.Dial("tcp", dest)
					for conn == nil {
						if err != nil {
							fmt.Println(err)
						}
						time.Sleep(time.Second)
						conn, err = net.Dial("tcp", dest)
					}

					go bootstrapper.handleConnection(conn, bootstrapMsg)
				}(peerAddr)
			}

			go bootstrapper.checkReceivedPeerInfos(done)
		}()

		<-done
		err = bootstrapper.persistPeerInfos()
		if err != nil {
			panic(err)
		}
		updateConfig()
	},
}

var channel = &cobra.Command{
	Use:   "channel",
	Short: "generate a channel id for bootstrapping",
	Run: func(cmd *cobra.Command, args []string) {
		channelId, err := rand.Int(rand.Reader, big.NewInt(999))
		if err != nil {
			panic(err)
		}
		expireTime := time.Now().Add(30 * time.Minute).Unix()
		fmt.Printf("channel id: %s\n", fmt.Sprintf("%.3d%s", channelId.Int64(), convertTimestampToHex(expireTime)))
	},
}

func askChannelId() string {
	reader := bufio.NewReader(os.Stdin)
	channelId, err := GetString("please set channel id of this session", reader)
	if err != nil {
		panic(err)
	}
	return channelId
}

func askChannelPasswd() string {
	if p, err := speakeasy.Ask("please input password to secure secret bootstrap session:"); err == nil {
		if p2, err := speakeasy.Ask("please input again:"); err == nil {
			if p2 != p {
				panic(fmt.Errorf("two inputs does not match, please start again"))
			} else {
				return p
			}
		} else {
			panic(err)
		}
	} else {
		panic(err)
	}
	return ""
}

func setN() {
	if common.TssCfg.Parties > 0 {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	n, err := GetInt("please set total parties(n): ", reader)
	if err != nil {
		panic(err)
	}
	if n <= 1 {
		panic(fmt.Errorf("n should greater than 1"))
	}
	common.TssCfg.Parties = n
}

func askPeerAddrs() []string {
	reader := bufio.NewReader(os.Stdin)
	peerAddrs := make([]string, 0, common.TssCfg.Parties-1)

	for i := 1; i < common.TssCfg.Parties; i++ {
		ithParty := humanize.Ordinal(i)
		addr, err := GetString(fmt.Sprintf("please input peer listen address of the %s party (e.g. /ip4/127.0.0.1/tcp/27148)", ithParty), reader)
		if err != nil {
			panic(err)
		}
		peerAddrs = append(peerAddrs, addr)
	}
	return peerAddrs
}

func (b *bootstrapper) handleConnection(conn net.Conn, msg *BootstrapMessage) {
	conn.SetWriteDeadline(time.Now().Add(time.Second))
	encoder := gob.NewEncoder(conn)
	if err := encoder.Encode(msg); err != nil {
		panic(err)
	}

	conn.SetReadDeadline(time.Now().Add(time.Second))
	decoder := gob.NewDecoder(conn)
	var peerMsg BootstrapMessage
	if err := decoder.Decode(&peerMsg); err != nil {
		// TODO: handle error
	} else {
		if err := b.handleBootstrapMsg(peerMsg); err != nil {
			panic(err)
		}
	}
	conn.Close()
}

type BootstrapMessage struct {
	ChannelId string // channel id + epoch timestamp in dex
	PeerInfo  []byte // encrypted moniker+libp2pid
	Addr      string
}

type peerInfo struct {
	id         string
	moniker    string
	remoteAddr string
}

func NewBootstrapMessage(channelId, passphrase, moniker string, id common.TssClientId, addr string) (*BootstrapMessage, error) {
	pi, err := encrypt(passphrase, channelId, moniker, string(id))
	if err != nil {
		return nil, err
	}
	return &BootstrapMessage{
		ChannelId: channelId,
		PeerInfo:  pi,
		Addr:      addr,
	}, nil
}

func (b *bootstrapper) handleBootstrapMsg(peerMsg BootstrapMessage) error {
	if moniker, id, err := decrypt(peerMsg.PeerInfo, b.channelId, b.channelPasswd); err != nil {
		return err
	} else {
		if info, ok := b.peers.Load(id); info != nil && ok {
			if moniker != info.(peerInfo).moniker {
				return fmt.Errorf("received different moniker for id: %s", id)
			}
		} else {
			pi := peerInfo{
				id:         id,
				moniker:    moniker,
				remoteAddr: peerMsg.Addr,
			}
			b.peers.Store(id, pi)
		}
	}
	return nil
}

func (b *bootstrapper) checkReceivedPeerInfos(done chan<- bool) {
	for {
		received := 0
		b.peers.Range(func(_, _ interface{}) bool {
			received++
			return true
		})
		if received == len(b.peerAddrs) {
			done <- true
			break
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (b bootstrapper) persistPeerInfos() error {
	peerAddrs := make([]string, 0)
	expectedPeers := make([]string, 0)
	var err error
	b.peers.Range(func(id, value interface{}) bool {
		if pi, ok := value.(peerInfo); ok {
			peerAddrs = append(peerAddrs, pi.remoteAddr)
			expectedPeers = append(expectedPeers, fmt.Sprintf("%s@%s", pi.moniker, pi.id))
			return true
		} else {
			err = fmt.Errorf("failed to parse peerInfo from received messages")
			return false
		}
	})

	if err != nil {
		return err
	}

	common.TssCfg.PeerAddrs = peerAddrs
	common.TssCfg.ExpectedPeers = expectedPeers
	return nil
}

// conversion between hex and int32 epoch seconds
// refer: https://www.epochconverter.com/hex
func convertTimestampToHex(timestamp int64) string {
	buf := bytes.Buffer{}
	if err := binary.Write(&buf, binary.BigEndian, int32(timestamp)); err != nil {
		return ""
	}
	return fmt.Sprintf("%X", buf.Bytes())
}

// conversion between hex and int32 epoch seconds
// refer: https://www.epochconverter.com/hex
func convertHexToTimestamp(hexTimestamp string) int {
	dst := make([]byte, 8)
	hex.Decode(dst, []byte(hexTimestamp))
	var epochSeconds int32
	if err := binary.Read(bytes.NewReader(dst), binary.BigEndian, &epochSeconds); err != nil {
		return math.MaxInt64
	}
	return int(epochSeconds)
}

func encrypt(passphrase, channelId, moniker, id string) ([]byte, error) {
	text := []byte(fmt.Sprintf("%s@%s@%s", channelId, moniker, id))
	key := sha256.Sum256([]byte(passphrase))

	// generate a new aes cipher using our 32 byte long key
	c, err := aes.NewCipher(key[:])
	// if there are any errors, handle them
	if err != nil {
		return nil, err
	}

	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	// if any error generating new GCM
	// handle them
	if err != nil {
		return nil, err
	}

	// creates a new byte array the size of the nonce
	// which must be passed to Seal
	nonce := make([]byte, gcm.NonceSize())
	// populates our nonce with a cryptographically secure
	// random sequence
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// here we encrypt our text using the Seal function
	// Seal encrypts and authenticates plaintext, authenticates the
	// additional data and appends the result to dst, returning the updated
	// slice. The nonce must be NonceSize() bytes long and unique for all
	// time, for a given key.
	return gcm.Seal(nonce, nonce, text, nil), nil
}

func decrypt(ciphertext []byte, channelId, passphrase string) (moniker, id string, error error) {
	key := sha256.Sum256([]byte(passphrase))
	c, err := aes.NewCipher(key[:])
	if err != nil {
		error = err
		return
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		error = err
		return
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		error = fmt.Errorf("ciphertext is not as long as expected")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		error = err
		return
	}

	res := strings.SplitN(string(plaintext), "@", 3)
	if len(res) != 3 {
		error = fmt.Errorf("wrong format of decrypted plaintext")
		return
	}
	if res[0] != channelId {
		error = fmt.Errorf("wrong channel id of message")
	}
	epochSeconds := convertHexToTimestamp(channelId[3:])
	if time.Now().Unix() > int64(epochSeconds) {
		error = fmt.Errorf("password has been expired")
		return
	}
	return res[1], res[2], nil
}

func convertMultiAddrStrToNormalAddr(listenAddr string) (string, error) {
	re := regexp.MustCompile(`((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))\/tcp\/([0-9]+)`)
	all := re.FindStringSubmatch(listenAddr)
	if len(all) != 6 {
		return "", fmt.Errorf("failed to convert multiaddr to listen addr")
	}
	return fmt.Sprintf("%s:%s", all[1], all[5]), nil
}
