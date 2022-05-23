package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/Safulet/tss/client"
	"github.com/Safulet/tss/common"
	"github.com/Safulet/tss/p2p"
	"github.com/Safulet/tss/ssdp"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/whyrusleeping/go-logging"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

func init() {
	rootCmd.AddCommand(recoverCmd)
}

var recoverCmd = &cobra.Command{
	Use:   "recover",
	Short: "recover in one single cmd",
	Long:  "recover in one single cmd, follow the input tips",
	PreRun: func(cmd *cobra.Command, args []string) {
		askParties()
		askThreshold()
		askChannel()
		askPassphrase()
		common.TssCfg.LogLevel = logging.INFO.String() // no configs here, mandatory set log level
		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		filenames := args

		for idx, filename := range filenames {
			reloadTssClient(filename, idx)
		}
	},
}

func askParties() {
	if parties := viper.GetString("parties"); parties != "" {
		common.TssCfg.Parties, _ = strconv.Atoi(parties)
		return
	}

	reader := bufio.NewReader(os.Stdin)
	parties, err := common.GetInt("please set parties of the recover phase: ", 0, reader)
	if err != nil {
		common.Panic(err)
	}
	viper.Set("parties", parties)
	common.TssCfg.Parties = parties
}

func askThreshold() {
	if threshold := viper.GetString("threshold"); threshold != "" {
		common.TssCfg.Threshold, _ = strconv.Atoi(threshold)
		return
	}

	reader := bufio.NewReader(os.Stdin)
	threshold, err := common.GetInt("please set threshold of the recover phase: ", 0, reader)
	if err != nil {
		common.Panic(err)
	}
	viper.Set("threshold", threshold)
	common.TssCfg.Threshold = threshold
}

func reloadTssClient(filename string, idx int) *client.TssClient {
	curve, _ := tss.GetCurveName(tss.EC())
	recoveredData := reloadLocalPartySaveData(filename, string(curve))
	recoveredConfig := recoverConfig(recoveredData)

	checkPrerequisites(recoveredData, recoveredConfig)

	recoverHomeDir(recoveredConfig)
	recoverP2pKey(recoveredConfig)
	recoverListernAddresses(recoveredConfig, idx)
	recoverBootstrap(recoveredConfig)

	common.TssCfg = *recoveredConfig
	common.TssCfg.Password = viper.GetString("password")
	common.TssCfg.KDFConfig = common.DefaultKDFConfig()
	common.TssCfg.LogLevel = logging.INFO.String()
	updateConfig()

	c := client.NewRecoverTssClient(&common.TssCfg, recoveredData)
	c.RecoverKeygen(recoveredData)
	return c
}

func checkPrerequisites(data *keygen.LocalPartySaveData, config *common.TssConfig) {
	if len(data.Ks) != config.Parties {
		common.Panic(errors.New("parties number not match"))
	}
	if !data.Validate() {
		common.Panic(errors.Errorf("local party data error, shareId is %s", config.Id))
	}
}

func reloadLocalPartySaveData(moniker string, curve string) *keygen.LocalPartySaveData {
	src, err := os.OpenFile(path.Join(moniker), os.O_RDONLY, 0400)
	if err != nil {
		common.Panic(err)
	}
	defer src.Close()

	sBytes, err := ioutil.ReadAll(src)
	if err != nil {
		common.Panic(err)
	}
	dataMap := make(map[string]interface{})
	err = json.Unmarshal(sBytes, &dataMap)
	if err != nil {
		common.Panic(err)
	}
	var result keygen.LocalPartySaveData

	for k, v := range dataMap {
		if strings.Contains(k, "sensitiveData") {
			for k, v := range v.(map[string]interface{}) {
				if strings.Contains(k, curve) && v != nil && len(v.(string)) != 0 {
					d := json.NewDecoder(strings.NewReader(v.(string)))
					d.UseNumber()
					err = d.Decode(&result)
					if err != nil {
						common.Panic(err)
					}
					words := strings.Split(k, "-")
					vault := words[len(words)-1]
					common.TssCfg.Vault = vault
					client.Logger.Infof("vault is %s", vault)
					break
				}
			}
		}
	}

	return &result
}

func recoverP2pKey(config *common.TssConfig) {
	privKey, id, err := p2p.NewP2pPrivKey()
	if err != nil {
		common.Panic(err)
	}

	bytes, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		common.Panic(err)
	}
	if err := ioutil.WriteFile(path.Join(config.Home, config.Vault, "node_key"), bytes, os.FileMode(0600)); err != nil {
		common.Panic(err)
	}

	config.Id = common.TssClientId(id.String())
}

func recoverHomeDir(config *common.TssConfig) {
	home := viper.GetString(flagHome)
	config.Home = home
	makeHomeDir(config.Home, config.Vault)
	client.Logger.Infof("the restore home dir is %s/%s", config.Home, config.Vault)
}

var candidates []int

func recoverListernAddresses(config *common.TssConfig, idx int) {
	if candidates == nil {
		ports, err := freeport.GetFreePorts(common.TssCfg.Threshold + 1)
		if err != nil {
			common.Panic(err)
		}
		candidates = ports
	}

	config.ListenAddr = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", candidates[idx])
}

func recoverConfig(data *keygen.LocalPartySaveData) *common.TssConfig {
	config := common.TssConfig{}
	config.Parties = common.TssCfg.Parties
	config.Threshold = common.TssCfg.Threshold
	config.Vault = common.TssCfg.Vault
	config.ChannelId = common.TssCfg.ChannelId
	config.ChannelPassword = common.TssCfg.ChannelPassword
	config.Message = "0"
	config.Moniker = data.ShareID.String()
	return &config
}

func recoverBootstrap(config *common.TssConfig) {
	src, err := common.ConvertMultiAddrStrToNormalAddr(config.ListenAddr)
	if err != nil {
		common.Panic(err)
	}
	listenAddrs := getListenAddrs(config.ListenAddr)
	numOfPeers := config.Parties - 1
	bootstrapper := common.NewBootstrapper(numOfPeers, config)
	listener, err := net.Listen("tcp", src)
	if err != nil {
		common.Panic(err)
	}
	defer func() {
		err = listener.Close()
		if err != nil {
			client.Logger.Error(err)
		}
		client.Logger.Info("closed ssdp listener")
	}()

	done := make(chan bool)
	go acceptConnRoutine(listener, bootstrapper, done)

	peerAddrs := recoverPeerAddrsViaSsdp(numOfPeers, listenAddrs, config)
	client.Logger.Debugf("Found peers via ssdp: %v", peerAddrs)

	go func() {
		for _, peerAddr := range peerAddrs {
			go func(peerAddr string) {
				dest, err := common.ConvertMultiAddrStrToNormalAddr(peerAddr)
				if err != nil {
					common.Panic(fmt.Errorf("failed to convert peer multiAddr to addr: %v", err))
				}
				client.Logger.Debugf("going to dial: %s", peerAddr)
				conn, err := net.Dial("tcp", dest)
				for conn == nil {
					if err != nil {
						if !strings.Contains(err.Error(), "connection refused") {
							client.Logger.Errorf("dial failed: %v", err)
							common.Panic(err)
						}
					}
					time.Sleep(time.Second)
					conn, err = net.Dial("tcp", dest)
				}
				client.Logger.Debugf("done dial: %s", peerAddr)
				defer conn.Close()
				handleConnection(conn, bootstrapper)
			}(peerAddr)
		}

		checkReceivedPeerInfos(bootstrapper, done)
	}()

	<-done

	addresses := make([]string, 0)
	expectedPeers := make([]string, 0)
	bootstrapper.Peers.Range(func(id, value interface{}) bool {
		if pi, ok := value.(common.PeerInfo); ok {
			addresses = append(addresses, pi.RemoteAddr)
			expectedPeers = append(expectedPeers, fmt.Sprintf("%s@%s", pi.Moniker, pi.Id))
			return true
		} else {
			err = fmt.Errorf("failed to parse peerInfo from received messages")
			return false
		}
	})

	config.PeerAddrs, config.ExpectedPeers = mergeAndUpdate(
		config.PeerAddrs,
		config.ExpectedPeers,
		addresses,
		expectedPeers)
	if err != nil {
		common.Panic(err)
	}
}

func recoverPeerAddrsViaSsdp(n int, listenAddrs string, config *common.TssConfig) []string {
	ssdpSrv := ssdp.NewSsdpService(config.Moniker, config.Vault, listenAddrs, n, nil)
	ssdpSrv.CollectPeerAddrs()
	var peerAddrs []string
	ssdpSrv.PeerAddrs.Range(func(_, value interface{}) bool {
		if peerAddr, ok := value.(string); ok {
			peerAddrs = append(peerAddrs, peerAddr)
		}
		return true
	})
	return peerAddrs
}

func askChannel() {
	if channelId := viper.GetString("channel_id"); channelId == "" {
		setChannelId()
	} else {
		common.TssCfg.ChannelId = channelId
	}
	if channelPassWord := viper.GetString("channel_password"); channelPassWord == "" {
		setChannelPasswd()
	} else {
		common.TssCfg.ChannelPassword = channelPassWord
	}
}
