package cmd

import (
	"bufio"
	"crypto/rand"
	"encoding/gob"
	"fmt"
	"math/big"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bgentry/speakeasy"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/tss/common"
)

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
		channelId := setChannelId()
		setChannelPasswd()
		setN()
		numOfPeers := common.TssCfg.Parties - 1
		if common.TssCfg.BMode == common.PreRegroupMode {
			numOfPeers = common.TssCfg.UnknownParties
		}
		peerAddrs := askPeerAddrs(numOfPeers)

		bootstrapper := &common.Bootstrapper{
			ChannelId:       channelId,
			ChannelPassword: common.TssCfg.ChannelPassword,
			PeerAddrs:       peerAddrs,
			Cfg:             &common.TssCfg,
		}

		bootstrapMsg, err := common.NewBootstrapMessage(
			channelId,
			common.TssCfg.ChannelPassword,
			common.TssCfg.Moniker,
			common.TssCfg.Id,
			common.TssCfg.ListenAddr,
			false,
			false)
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
					return
				} else {
					fmt.Printf("%s connected to us!\n", conn.RemoteAddr().String())
					go handleConnection(conn, bootstrapper, bootstrapMsg)
				}
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

					go handleConnection(conn, bootstrapper, bootstrapMsg)
				}(peerAddr)
			}

			checkReceivedPeerInfos(bootstrapper, done)
		}()

		<-done
		err = persistPeerInfos(bootstrapper)
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
		fmt.Printf("channel id: %s\n", fmt.Sprintf("%.3d%s", channelId.Int64(), common.ConvertTimestampToHex(expireTime)))
	},
}

func setChannelId() string {
	if common.TssCfg.ChannelId != "" {
		return common.TssCfg.ChannelId
	}

	reader := bufio.NewReader(os.Stdin)
	channelId, err := GetString("please set channel id of this session", reader)
	if err != nil {
		panic(err)
	}
	common.TssCfg.ChannelId = channelId
	return channelId
}

func setChannelPasswd() {
	if common.TssCfg.ChannelPassword != "" {
		return
	}

	if p, err := speakeasy.Ask("please input password to secure secret bootstrap session:"); err == nil {
		common.TssCfg.ChannelPassword = p
	} else {
		panic(err)
	}
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

func askPeerAddrs(n int) []string {
	reader := bufio.NewReader(os.Stdin)
	peerAddrs := make([]string, 0, common.TssCfg.Parties-1)

	for i := 1; i <= n; i++ {
		ithParty := humanize.Ordinal(i)
		addr, err := GetString(fmt.Sprintf("please input peer listen address of the %s (out of %d) party (e.g. /ip4/127.0.0.1/tcp/27148)", ithParty, n), reader)
		if err != nil {
			panic(err)
		}
		peerAddrs = append(peerAddrs, addr)
	}
	return peerAddrs
}

func handleConnection(conn net.Conn, b *common.Bootstrapper, msg *common.BootstrapMessage) {
	// TODO: support ipv6
	realIp := strings.SplitN(conn.LocalAddr().String(), ":", 2)
	msgForConnect := common.BootstrapMessage{
		ChannelId: msg.ChannelId,
		PeerInfo:  msg.PeerInfo,
		Addr:      common.ReplaceIpInAddr(msg.Addr, realIp[0]),
		IsOld:     msg.IsOld,
		IsNew:     msg.IsNew,
	}
	conn.SetWriteDeadline(time.Now().Add(time.Second))
	encoder := gob.NewEncoder(conn)
	if err := encoder.Encode(msgForConnect); err != nil {
		panic(err)
	}

	conn.SetReadDeadline(time.Now().Add(time.Second))
	decoder := gob.NewDecoder(conn)
	var peerMsg common.BootstrapMessage
	if err := decoder.Decode(&peerMsg); err != nil {
		// TODO: handle error
	} else {
		if err := b.HandleBootstrapMsg(peerMsg); err != nil {
			panic(err)
		}
	}
	conn.Close()
}

func checkReceivedPeerInfos(bootstrapper *common.Bootstrapper, done chan<- bool) {
	for {
		if bootstrapper.IsFinished() {
			done <- true
			close(done)
			break
		} else {
			time.Sleep(time.Second)
		}
	}
}

func persistPeerInfos(bootstrapper *common.Bootstrapper) error {
	peerAddrs := make([]string, 0)
	expectedPeers := make([]string, 0)
	var err error
	bootstrapper.Peers.Range(func(id, value interface{}) bool {
		if pi, ok := value.(common.PeerInfo); ok {
			peerAddrs = append(peerAddrs, pi.RemoteAddr)
			expectedPeers = append(expectedPeers, fmt.Sprintf("%s@%s", pi.Moniker, pi.Id))
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

func convertMultiAddrStrToNormalAddr(listenAddr string) (string, error) {
	re := regexp.MustCompile(`((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))\/tcp\/([0-9]+)`)
	all := re.FindStringSubmatch(listenAddr)
	if len(all) != 6 {
		return "", fmt.Errorf("failed to convert multiaddr to listen addr")
	}
	return fmt.Sprintf("%s:%s", all[1], all[5]), nil
}
