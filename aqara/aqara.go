package aqara

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const nonceLength int = 16

type AqaraRegionServer string

const (
	ServerRegionChina      AqaraRegionServer = "open-cn.aqara.com"
	ServerRegionUSA        AqaraRegionServer = "open-usa.aqara.com"
	ServerRegionSouthKorea AqaraRegionServer = "open-kr.aqara.com"
	ServerRegionRussia     AqaraRegionServer = "open-ru.aqara.com"
	ServerRegionEurope     AqaraRegionServer = "open-ger.aqara.com"
	ServerRegionSingapore  AqaraRegionServer = "open-sg.aqara.com"
)

type AqaraRequest struct {
	Intent string    `json:"intent"`
	Data   AqaraData `json:"data"`
}

type AqaraData interface {
}

type AqaraResponse struct {
	Code          int         `json:"code"`
	RequestID     string      `json:"requestId"`
	Message       string      `json:"message"`
	MessageDetail string      `json:"messageDetail"`
	Result        AqaraResult `json:"result"`
}

type AqaraResult interface {
}

type AqaraClient struct {
	region AqaraRegionServer
	appID  string
	keyID  string
	appKey string
}

// New returns a new AqaraClient.
func New(region AqaraRegionServer, appID, keyID, appKey string) *AqaraClient {
	return &AqaraClient{
		region: region,
		appID:  appID,
		keyID:  keyID,
		appKey: appKey,
	}
}

// Auth will request a new authorization code for a given Aqara account.
func (a *AqaraClient) Auth(account string) {
	type Data struct {
		Account             string `json:"account"`
		AccountType         int    `json:"accountType"`
		AccessTokenValidity string `json:"accessTokenValidity"`
	}

	data := Data{
		Account:             account,
		AccountType:         0,
		AccessTokenValidity: "1h",
	}

	request := AqaraRequest{
		Intent: "config.auth.getAuthCode",
		Data:   data,
	}

	type Result struct {
		AuthCode string `json:"authCode"`
	}

	result := Result{}
	response := AqaraResponse{
		Result: result,
	}

	err := a.apiCall(request, &response)
	if err != nil {
		log.Printf("Failed to do auth request: %v", err)
	}
}

// apiCall sends request to the Aqara API with the provided AqaraRequest (intent).
// Response is updated in the provided AqaraResponse pointer.
func (a *AqaraClient) apiCall(aqaraRequest AqaraRequest, aqaraResponse *AqaraResponse) error {
	nonce := getNonce(nonceLength)
	timestamp := getTimestamp()

	// TODO: accessToken needs to be set for authenticated requests (intents).
	signature := a.sign("", nonce, timestamp)

	const apiEndpoint = "/v3.0/open/api"
	url := fmt.Sprintf("https://%s%s", a.region, apiEndpoint)

	requestBody, err := json.Marshal(aqaraRequest)
	if err != nil {
		log.Printf("Failed to marshal request: %v", err)
		return err
	}

	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		log.Printf("Failed to prepare request: %v", err)
		return err
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Appid", a.appID)
	request.Header.Add("Keyid", a.keyID)
	request.Header.Add("Nonce", nonce)
	request.Header.Add("Time", timestamp)
	request.Header.Add("Sign", signature)
	request.Header.Add("Lang", "en")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Printf("Failed to do request: %v", err)
		return err
	}

	if response.StatusCode == http.StatusOK {
		log.Printf("Call to %q successful", url)
		responseBody, err := io.ReadAll(response.Body)
		defer response.Body.Close()
		if err != nil {
			log.Printf("Failed to get response body: %v", err)
			return err
		}

		err = json.Unmarshal(responseBody, aqaraResponse)
		if err != nil {
			log.Printf("Failed to unmarshal response: %v", err)
			return err
		}

		if aqaraResponse.Code != 0 {
			log.Printf("Aqara response code %v received with message %v", aqaraResponse.Code, aqaraResponse.MessageDetail)
			return fmt.Errorf("request against Aqara API failed with code: %v", aqaraResponse.Code)
		}

		return nil
	} else {
		log.Printf("Status %v received", response.StatusCode)
		return fmt.Errorf("failed to do request: %v", response.StatusCode)
	}
}

// sign calculates the signature that is expected in the Sign header.
func (a *AqaraClient) sign(accessToken, nonce, timestamp string) string {
	var s string
	if len(accessToken) != 0 {
		s = fmt.Sprintf("Accesstoken=%s&Appid=%s&Keyid=%s&Nonce=%s&Time=%s%s", accessToken, a.appID, a.keyID, nonce, timestamp, a.appKey)
	} else {
		s = fmt.Sprintf("Appid=%s&Keyid=%s&Nonce=%s&Time=%s%s", a.appID, a.keyID, nonce, timestamp, a.appKey)
	}
	s = strings.ToLower(s)

	hash := md5.Sum([]byte(s))

	return hex.EncodeToString(hash[:])
}

// getNonce returns a random string with a certain length.
func getNonce(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}

	return string(b)
}

// getTimestamp returns the current time in milliseconds as string.
func getTimestamp() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}
