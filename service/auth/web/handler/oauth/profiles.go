package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/micro/go-micro/v2/auth"
)

type googleProfile struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"given_name"`
	LastName  string `json:"family_name"`
	Picture   string `json:"picture"`
}

func getGoogleProfile(token string) (*auth.Account, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v1/userinfo?oauth_token=" + token)
	if err != nil {
		return nil, fmt.Errorf("Error getting account from Google: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error getting account from Google. Status: %v", resp.Status)
	}

	// Decode the users profile
	var p *googleProfile
	json.NewDecoder(resp.Body).Decode(&p)

	// construct the account
	return &auth.Account{
		ID:   p.ID,
		Type: "user",
	}, nil
}
