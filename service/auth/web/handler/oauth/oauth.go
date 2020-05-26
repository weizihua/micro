package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/auth/provider"
	"github.com/micro/go-micro/v2/auth/provider/oauth"
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/store"
	"github.com/micro/micro/v2/client/web"
	inauth "github.com/micro/micro/v2/internal/auth"
)

// NewHandler returns an initialised handler for the given oauth provider
func NewHandler(name string, opts *provider.Options, auth auth.Auth, store store.Store) *Handler {
	// get the auth endpoints from the predefined list.
	endpoints := inauth.OauthProviders[name]

	provider := oauth.NewProvider(
		provider.Scope(opts.Scope),
		provider.AuthURL(endpoints.AuthURL),
		provider.TokenURL(endpoints.TokenURL),
		provider.AuthStyle(endpoints.AuthStyle),
		provider.Credentials(opts.ClientID, opts.ClientSecret),
	)

	return &Handler{name, provider, auth, store}
}

// Handler impements the http handler funcs for an oauth provider
type Handler struct {
	name     string
	provider provider.Provider
	auth     auth.Auth
	store    store.Store
}

// Login generates the oauth state and redirects to the login endpoint for the given provider
func (h *Handler) Login(w http.ResponseWriter, req *http.Request) {
	state, err := h.generateOauthState()
	if err != nil {
		h.handleError(w, req, err.Error())
		return
	}

	params := make(url.Values)
	params.Add("response_type", "code")
	params.Add("state", state)
	params.Add("redirect_uri", h.redirectURI(req))

	if clientID := h.provider.Options().ClientID; len(clientID) > 0 {
		params.Add("client_id", clientID)
	}

	if scope := h.provider.Options().Scope; len(scope) > 0 {
		params.Add("scope", scope)
	}

	endpoint := fmt.Sprintf("%v?%v", h.provider.Options().AuthURL, params.Encode())
	http.Redirect(w, req, endpoint, http.StatusFound)
}

// Verify validates the oauth state, exchanges the token for the users profile and then creates
// an auth account for the user
func (h *Handler) Verify(w http.ResponseWriter, req *http.Request) {
	// validate the oauth state
	valid, err := h.validateOauthState(req.FormValue("state"))
	if err != nil {
		h.handleError(w, req, err.Error())
		return
	} else if !valid {
		h.handleError(w, req, "Invalid Oauth State")
		return
	}

	// perform the oauth call to exchange the code for an access token
	resp, err := http.PostForm(h.provider.Options().TokenURL, url.Values{
		"client_id":     {h.provider.Options().ClientID},
		"client_secret": {h.provider.Options().ClientSecret},
		"redirect_uri":  {h.redirectURI(req)},
		"grant_type":    {"authorization_code"},
		"code":          {req.FormValue("code")},
	})
	if err != nil {
		h.handleError(w, req, "Error getting access token from the provider: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		h.handleError(w, req, "Error getting access token from the provider. Status: %v", resp.Status)
	}

	// Decode the token
	var result struct {
		Token string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		h.handleError(w, req, "Error decoding token response: %v", err)
	}

	// Get the profile for the user. Each provider will have a different endpoint for this so we'll
	// add an implementation for each of the supported providers.
	var acc *auth.Account
	switch h.name {
	case "google":
		acc, _ = getGoogleProfile(result.Token)
	}
	fmt.Println(result.Token)
}

func (h *Handler) handleError(w http.ResponseWriter, req *http.Request, format string, args ...interface{}) {
	logger.Errorf(format, args...)
	params := url.Values{"error": {fmt.Sprintf(format, args...)}}
	http.Redirect(w, req, "/?"+params.Encode(), http.StatusFound)
}

func (h *Handler) loginUser(w http.ResponseWriter, req *http.Request, tok *auth.Token) {
	http.SetCookie(w, &http.Cookie{
		Name:  auth.TokenCookieName,
		Value: tok.AccessToken,
		Path:  "/",
	})

	http.Redirect(w, req, "/", http.StatusFound)
}

func (h *Handler) redirectURI(req *http.Request) string {
	scheme := "https"
	if req.TLS == nil {
		scheme = "http"
	}

	host := req.Header.Get(web.HostHeader)     // e.g. micro.mu
	base := req.Header.Get(web.BasePathHeader) // e.g. /auth

	return fmt.Sprintf("%v://%v%v/%v/verify", scheme, host, base, h.name) // e.g. https://micro.mu/auth/google/verify
}
