package cmd

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/Safulet/chain-integration/v2/blockchain"
	"github.com/Safulet/chain-integration/v2/proto/ecdsa/signing"
	"github.com/Safulet/tss/client"
	"github.com/shopspring/decimal"
	"strings"

	"github.com/libp2p/go-libp2p"
	"github.com/multiformats/go-multiaddr"

	common2 "github.com/Safulet/chain-integration/v2/blockchain/common"
	"github.com/Safulet/tss/common"
)

func getListenAddrs(listenAddr string) string {
	addr, err := multiaddr.NewMultiaddr(listenAddr)
	if err != nil {
		common.Panic(err)
	}
	host, err := libp2p.New(context.Background(), libp2p.ListenAddrs(addr))
	if err != nil {
		common.Panic(err)
	}

	builder := strings.Builder{}
	for i, addr := range host.Addrs() {
		if i > 0 {
			fmt.Fprint(&builder, ", ")
		}
		fmt.Fprintf(&builder, "%s", addr)
	}
	host.Close()
	return builder.String()
}

func generateTransfer(network, from, to string, amount, asset, gasPrice string, nonce int64) common.Transfer {
	return common.Transfer{
		Network:  network,
		From:     from,
		To:       to,
		Amount:   amount,
		Asset:    asset,
		GasPrice: gasPrice,
		Nonce:    nonce,
	}
}

func amountWithDecimal(transfer *common.Transfer) string {
	amountF, err := decimal.NewFromString(transfer.Amount)
	if err != nil {
		return transfer.Amount
	}
	if need, decimal := needCalcAmountWithDecimal(transfer.GetNetwork()); need && decimal != 0 {
		return common2.CalcAmountWithDecimal(amountF, decimal).String()
	}
	return transfer.Amount
}

func needCalcAmountWithDecimal(network string) (bool, int32) {
	if strings.EqualFold(network, common.Solana) {
		return true, common.SolanaDecimal
	}
	return false, 0
}

func Broadcast(bc blockchain.Blockchain, message *common.SignatureMessage) (string, error) {
	signatures := make([]signing.SignatureData, 0, len(message.Signatures))
	for _, s := range message.Signatures {
		sig := signing.SignatureData{
			Signature:         s.Signature,
			SignatureRecovery: s.SignatureRecovery,
			R:                 s.R,
			S:                 s.S,
			M:                 s.M,
		}
		signatures = append(signatures, sig)
	}
	tx, err := bc.BuildTransaction(signatures)
	if err != nil {
		return "", err
	}

	txInHex := "0x" + hex.EncodeToString(tx)
	client.Logger.Infof(txInHex)
	hash, err := bc.Broadcast(tx)

	if err == nil {
		return hash, nil
	} else {
		return "", err // blockchain error should not be regarded as broadcast error, so that manager can save the status and reason
	}
}
