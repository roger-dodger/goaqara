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
	region  = flag.String("region", "europe", "Aqara server region: china, usa, southkorea, russia, europe, singapore")
	account = flag.String("account", "", "Aqara registered phone number or email address")
	debug   = flag.Bool("debug", false, "enable debug output")
)

func main() {
	flag.Parse()

	var serverRegion aqara.AqaraRegionServer

	switch *region {
	case "china":
		serverRegion = aqara.ServerRegionChina
	case "usa":
		serverRegion = aqara.ServerRegionUSA
	case "southkorea":
		serverRegion = aqara.ServerRegionSouthKorea
	case "russia":
		serverRegion = aqara.ServerRegionRussia
	case "europe":
		serverRegion = aqara.ServerRegionEurope
	case "singapore":
		serverRegion = aqara.ServerRegionSingapore
	default:
		fmt.Println("No valid server region provided. Defaulting to 'europe'.")
		serverRegion = aqara.ServerRegionEurope
	}

	if *appID == "" || *keyID == "" || *appKey == "" || *account == "" {
		fmt.Println("You must provide the following arguments: appid, keyid, appkey and account")
		os.Exit(-1)
	}

	aqaraClient := aqara.New(serverRegion, *appID, *keyID, *appKey, *account, *debug)
	aqaraClient.GetAuthCode()

	fmt.Print("Enter auth code sent via SMS or email: ")
	var authCode string
	fmt.Scanln(&authCode)

	aqaraClient.GetToken(authCode)

	aqaraClient.GetDevices()
}
