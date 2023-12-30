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
	Intent string      `json:"intent"`
	Data   interface{} `json:"data"`
}

type AqaraResponse struct {
	Code          int             `json:"code"`
	RequestID     string          `json:"requestId"`
	Message       string          `json:"message"`
	MessageDetail string          `json:"messageDetail"`
	Result        json.RawMessage `json:"result"`
}

type AqaraClient struct {
	region       AqaraRegionServer
	appID        string
	keyID        string
	appKey       string
	account      string
	accessToken  string
	refreshToken string
	debug        bool
}

// New returns a new AqaraClient.
func New(region AqaraRegionServer, appID, keyID, appKey, account string, debug bool) *AqaraClient {
	return &AqaraClient{
		region:       region,
		appID:        appID,
		keyID:        keyID,
		appKey:       appKey,
		account:      account,
		accessToken:  "", // updated after login
		refreshToken: "", // updated after login
		debug:        debug,
	}
}

// GetAuthCode will request a new authorization code for a given Aqara account.
func (a *AqaraClient) GetAuthCode() {
	type Data struct {
		Account             string `json:"account"`
		AccountType         int    `json:"accountType"`
		AccessTokenValidity string `json:"accessTokenValidity"`
	}

	request := AqaraRequest{
		Intent: "config.auth.getAuthCode",
		Data: Data{
			Account:             a.account,
			AccountType:         0,
			AccessTokenValidity: "1h",
		},
	}

	response := AqaraResponse{}

	if err := a.apiCall(request, &response, false); err != nil {
		log.Printf("Failed to do auth request: %v", err)
	}
}

// GetToken exchanges the authorization code for an access token.
func (a *AqaraClient) GetToken(authCode string) {
	type Data struct {
		AuthCode    string `json:"authCode"`
		Account     string `json:"account"`
		AccountType int    `json:"accountType"`
	}

	request := AqaraRequest{
		Intent: "config.auth.getToken",
		Data: Data{
			AuthCode:    authCode,
			Account:     a.account,
			AccountType: 0,
		},
	}

	response := AqaraResponse{}

	if err := a.apiCall(request, &response, false); err != nil {
		log.Printf("Failed to do token request: %v", err)
	}

	if response.Code == 0 {
		log.Printf("Login successful, updating account information")

		type Result struct {
			ExpiresIn    string `json:"expiresIn"`
			OpenID       string `json:"openId"`
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
		}

		var result Result
		if err := json.Unmarshal(response.Result, &result); err != nil {
			log.Printf("Failed to unmarshal result: %v", err)
		}

		a.accessToken = result.AccessToken
		a.refreshToken = result.RefreshToken
	}
}

// GetDevices retreives all devices for a certain account.
func (a *AqaraClient) GetDevices() {
	type Data struct {
		DeviceIDs  []string `json:"dids"`
		PositionID string   `json:"positionId"`
		PageNum    int      `json:"pageNum"`
		PageSize   int      `json:"pageSize"`
	}

	request := AqaraRequest{
		Intent: "query.device.info",
		Data: Data{
			DeviceIDs:  []string{},
			PositionID: "",
			PageNum:    1,
			PageSize:   100,
		},
	}

	response := AqaraResponse{}

	if err := a.apiCall(request, &response, true); err != nil {
		log.Printf("Failed query devices: %v", err)
	}

	if response.Code == 0 {
		type Device struct {
			DID             string `json:"did"`
			ParentDID       string `json:"parentDid"`
			PositionID      string `json:"positionId"`
			CreateTime      string `json:"createTime"`
			UpdateTime      string `json:"updateTime"`
			Model           string `json:"model"`
			ModelType       int    `json:"modelType"`
			State           int    `json:"state"`
			FirmwareVersion string `json:"firmwareVersion"`
			DeviceName      string `json:"deviceName"`
			TimeZone        string `json:"timeZone"`
		}

		type Result struct {
			Data       []Device `json:"data"`
			TotalCount int      `json:"totalCount"`
		}

		var result Result
		if err := json.Unmarshal(response.Result, &result); err != nil {
			log.Printf("Failed to unmarshal result: %v", err)
		}

		log.Printf("Number of devices received: %v", result.TotalCount)
		for _, device := range result.Data {
			fmt.Printf("Device Name:  %v", device.DeviceName)
			fmt.Printf("Device Model: %v", device.Model)
		}
	}
}

// apiCall sends request to the Aqara API with the provided AqaraRequest (intent).
// Response is updated in the provided AqaraResponse pointer.
func (a *AqaraClient) apiCall(aqaraRequest AqaraRequest, aqaraResponse *AqaraResponse, authenticated bool) error {

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

	nonce := getNonce(nonceLength)
	timestamp := getTimestamp()
	var signature string
	if authenticated {
		request.Header.Add("Accesstoken", a.accessToken)
		signature = a.sign(a.accessToken, nonce, timestamp)
	} else {
		signature = a.sign("", nonce, timestamp)
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
			log.Printf("Aqara response with code %v received with message %v", aqaraResponse.Code, aqaraResponse.MessageDetail)
			return fmt.Errorf("request against Aqara API failed with code: %v", aqaraResponse.Code)
		}

		if a.debug {
			log.Printf("**DEBUG**: %v", string(responseBody))
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
