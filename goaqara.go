package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/roger-dodger/goaqara/aqara"
)

var (
	appID   = flag.String("appid", "", "Aqara App ID")
	keyID   = flag.String("keyid", "", "Aqara Key ID")
	appKey  = flag.String("appkey", "", "Aqara App Key")
	account = flag.String("account", "", "Aqara registered phone number or email address")
)

func main() {
	flag.Parse()

	if *appID == "" || *keyID == "" || *appKey == "" || *account == "" {
		fmt.Println("You must provide the following arguments: appid, keyid, appkey and account")
		os.Exit(-1)
	}

	aqaraClient := aqara.New(aqara.ServerRegionEurope, *appID, *keyID, *appKey)
	aqaraClient.Auth(*account)
}
