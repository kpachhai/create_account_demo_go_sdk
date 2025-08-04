package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	hedera "github.com/hiero-ledger/hiero-sdk-go/v2/sdk"
)

func main() {
    // 1. load your operator credentials
    operatorId, err := hedera.AccountIDFromString(os.Getenv("OPERATOR_ID"))
    if err != nil {
        panic(err)
    }

    operatorKey, err := hedera.PrivateKeyFromString(os.Getenv("OPERATOR_KEY"))
    if err != nil {
        panic(err)
    }

    // 2. initialize the client for testnet
    client := hedera.ClientForTestnet()
    client.SetOperator(operatorId, operatorKey)

    // 3. generate a new key pair
    newPrivateKey, err := hedera.PrivateKeyGenerateEcdsa()
    if err != nil {
        panic(err)
    }
    newPublicKey := newPrivateKey.PublicKey()

    // 4. build & execute the account creation transaction
    transaction := hedera.NewAccountCreateTransaction().
        // set the account key with alias
        SetECDSAKeyWithAlias(newPublicKey).          
        SetInitialBalance(hedera.NewHbar(20)) // fund with 20 HBAR

    txResponse, err := transaction.Execute(client)
    if err != nil {
        panic(err)
    }

    receipt, err := txResponse.GetReceipt(client)
    if err != nil {
        panic(err)
    }

    newAccountId := *receipt.AccountID

    fmt.Printf("Hedera account created: %s\n", newAccountId.String())
    fmt.Printf("EVM Address: 0x%s\n", newPublicKey.ToEvmAddress())

    // Wait for Mirror Node to populate data
    fmt.Println("Waiting for Mirror Node to update...\n")
    time.Sleep(6 * time.Second)

    // 5. query balance using Mirror Node
    mirrorNodeUrl := "https://testnet.mirrornode.hedera.com/api/v1/balances?account.id=" + newAccountId.String()

    resp, err := http.Get(mirrorNodeUrl)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        panic(err)
    }

    var data struct {
        Balances []struct {
            Balance int64 `json:"balance"`
        } `json:"balances"`
    }

    err = json.Unmarshal(body, &data)
    if err != nil {
        panic(err)
    }

    if len(data.Balances) > 0 {
        balanceInTinybars := data.Balances[0].Balance
        balanceInHbar := float64(balanceInTinybars) / 100000000.0
        
        fmt.Printf("Account balance: %.8f ℏ", balanceInHbar)
    } else {
        fmt.Println("Account balance not yet available in Mirror Node")
    }

    client.Close()
}
