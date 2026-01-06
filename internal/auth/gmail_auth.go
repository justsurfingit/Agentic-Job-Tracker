package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// GetClient retrieves a token, saves the token, then returns the generated client.
func GetGmailClient() *http.Client {
	// 1. Read credentials.json (The App's ID)
	b, err := os.ReadFile("credential.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// 2. Config with Scope (READONLY access to Gmail)
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// 3. Get the Client (The User's Session)
	client := getClient(config)
	return client
}

// getClient retrieves a token from a local file or prompts the user to login.
func getClient(config *oauth2.Config) *http.Client {
	// The file "token.json" stores the user's login session.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		// If file doesn't exist, we must login manually
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then return the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("\n---------------------------------------------------------\n")
	fmt.Printf("OPEN THIS LINK TO AUTHORIZE GMAIL ACCESS:\n%v\n", authURL)
	fmt.Printf("---------------------------------------------------------\n")
	fmt.Printf("After logging in, Google will give you a code (or check the URL bar localhost callback).\n")
	fmt.Printf("Paste the code here: ")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
