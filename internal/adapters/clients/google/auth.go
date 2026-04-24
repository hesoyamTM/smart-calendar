package google

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gcalendar "google.golang.org/api/calendar/v3"
)

func newOAuthConfig(cfg Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       []string{gcalendar.CalendarScope},
		Endpoint:     google.Endpoint,
	}
}

// AuthURL returns the Google OAuth2 authorization URL for the given state token.
func AuthURL(cfg Config, state string) string {
	return newOAuthConfig(cfg).AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// ExchangeCode exchanges an authorization code for an OAuth2 token.
func ExchangeCode(ctx context.Context, cfg Config, code string) (*oauth2.Token, error) {
	tok, err := newOAuthConfig(cfg).Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("oauth2 exchange: %w", err)
	}
	return tok, nil
}
