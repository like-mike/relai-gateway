package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	clientID := os.Getenv("AZURE_AD_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_AD_CLIENT_SECRET")
	tenantID := os.Getenv("AZURE_AD_TENANT_ID")

	// Get access token
	token, err := getAccessToken(tenantID, clientID, clientSecret)
	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}

	fmt.Println(token)

	// Call Graph API
	// err = getGroups(token)
	// if err != nil {
	// 	log.Fatalf("Failed to get groups: %v", err)
	// }

	err = getUserGroups(token, "dc5613bb-f448-4dca-9ebd-34ccc22b7acb")
	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}
}

func getAccessToken(tenantID, clientID, clientSecret string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("scope", "https://graph.microsoft.com/.default")

	url := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)
	resp, err := http.Post(url, "application/x-www-form-urlencoded", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("token request failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return "", err
	}
	return tokenResp.AccessToken, nil
}

func getUserGroups(accessToken, userID string) error {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/memberOf", userID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("graph request failed: %s", string(body))
	}

	var result struct {
		Value []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			OdataType   string `json:"@odata.type"`
		} `json:"value"`
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}

	fmt.Printf("Groups for user %s:\n", userID)
	for _, item := range result.Value {
		if item.OdataType == "#microsoft.graph.group" {
			fmt.Printf("- %s (%s)\n", item.DisplayName, item.ID)
		}
	}
	return nil
}

func getGroups(accessToken string) error {

	fmt.Println("got here")
	req, err := http.NewRequest("GET", "https://graph.microsoft.com/v1.0/groups", nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("graph request failed: %s", string(body))
	}

	var groupsResp struct {
		Value []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"value"`
	}
	err = json.Unmarshal(body, &groupsResp)
	if err != nil {
		return err
	}

	fmt.Println("Azure AD Groups:")
	for _, g := range groupsResp.Value {
		fmt.Printf("- %s (%s)\n", g.DisplayName, g.ID)
	}
	return nil
}
