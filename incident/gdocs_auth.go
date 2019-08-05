// Functions to authenticate with google docs.
package incident

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

type GoogleDriveConfig struct {
	CredentialsFileName string
	TokenFileName       string
	config              *oauth2.Config
}

// GDriveConfig is a global var to keep the google drive configuration
// yuck!
var GDriveConfig = &GoogleDriveConfig{}

// GetConfigs returns the oauth2 configuration
func (g *GoogleDriveConfig) GetConfig() (*oauth2.Config, error) {
	if g.config == nil {
		data, err := ioutil.ReadFile(g.CredentialsFileName)
		if err != nil {
			return nil, fmt.Errorf("Could not read the credentials file: %v", err)
		}
		// We use DriveScope (the widest possible) because more contained scopes didn't allow to copy
		// new docs easily. It's possible that DriveFileScope is enough though.
		config, err := google.ConfigFromJSON(data, drive.DriveScope)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse client secret file to config: %v", err)
		}
		g.config = config
	}
	return g.config, nil
}

// GetToken gets a token (either from file if present or via OAUTH2)
func (g *GoogleDriveConfig) GetToken() (*oauth2.Token, error) {
	_, err := g.GetConfig()
	if err != nil {
		return nil, err
	}
	tok, err := g.getTokenFromFile()
	if err != nil {
		tok, err = g.getTokenFromWeb()
		if err == nil {
			err = g.saveTokenToFile(tok)
		}
	}
	return tok, err
}

// GetClient returns an http client that can be used interacting with the GDocs API
func (g *GoogleDriveConfig) GetClient() (*http.Client, error) {
	tok, err := g.GetToken()
	if err != nil {
		return nil, err
	}
	return g.config.Client(context.Background(), tok), nil
}

func (g *GoogleDriveConfig) getTokenFromFile() (*oauth2.Token, error) {
	f, err := os.Open(g.TokenFileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func (g *GoogleDriveConfig) getTokenFromWeb() (*oauth2.Token, error) {
	authURL := g.config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Println("#### OAUTH2 AUTORIZATION FOR GOOGLE DOCS ####")
	fmt.Printf("Please go to the following link, then type the autorization code you obtain:\n%v\n", authURL)
	fmt.Printf("> ")
	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("Unable to read authorization code %v", err)
	}

	tok, err := g.config.Exchange(context.TODO(), authCode)
	if err != nil {
		err = fmt.Errorf("Unable to retrieve token from web %v", err)
	}
	return tok, err
}

// Save a token to file.
func (g *GoogleDriveConfig) saveTokenToFile(token *oauth2.Token) error {
	f, err := os.OpenFile(g.TokenFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}
