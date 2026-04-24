package v1

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hesoyamTM/smart-calendar/internal/models"
	"golang.org/x/oauth2"
)

// consumeState validates and removes a state nonce from the map.
// Returns false if the state is missing or expired.
func (c *Server) consumeState(m map[string]stateEntry, state string) bool {
	if state == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := m[state]
	if !ok || time.Now().After(entry.expiry) {
		delete(m, state)
		return false
	}
	delete(m, state)
	return true
}

// purgeExpired removes all expired entries from the map.
func (c *Server) purgeExpired(m map[string]stateEntry) {
	now := time.Now()
	for k, v := range m {
		if now.After(v.expiry) {
			delete(m, k)
		}
	}
}

func setTokenCookie(w http.ResponseWriter, name string, tok *oauth2.Token) error {
	data, err := json.Marshal(tok)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    hex.EncodeToString(data),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   cookieMaxAge,
	})
	return nil
}

func tokenFromCookie(r *http.Request, name string) (*oauth2.Token, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return nil, err
	}
	data, err := hex.DecodeString(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("decode token cookie: %w", err)
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("unmarshal token cookie: %w", err)
	}
	return &tok, nil
}

func chunkFrame(c models.Chunk) []byte {
	if c.Done {
		return []byte(`{"t":"e"}`)
	}
	v, _ := json.Marshal(c.Text)
	return append([]byte(`{"t":"c","v":`), append(v, '}')...)
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func timeNowAdd(d time.Duration) time.Time {
	return time.Now().Add(d)
}
