package v1

import (
	"net/http"

	google "github.com/hesoyamTM/smart-calendar/internal/adapters/clients/google"
)

func (c *Server) handleGoogleAuth(w http.ResponseWriter, r *http.Request) {
	state, err := randomState()
	if err != nil {
		http.Error(w, "failed to generate state", http.StatusInternalServerError)
		return
	}

	c.mu.Lock()
	c.purgeExpired(c.googleStates)
	c.googleStates[state] = stateEntry{expiry: timeNowAdd(stateExpiry)}
	c.mu.Unlock()

	http.Redirect(w, r, google.AuthURL(c.gcCfg, state), http.StatusTemporaryRedirect)
}

func (c *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if !c.consumeState(c.googleStates, r.URL.Query().Get("state")) {
		http.Error(w, "invalid or expired OAuth state", http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	tok, err := google.ExchangeCode(r.Context(), c.gcCfg, code)
	if err != nil {
		c.logger.Error("google oauth2 exchange failed", "error", err)
		http.Error(w, "Google authentication failed", http.StatusInternalServerError)
		return
	}

	if err := setTokenCookie(w, cookieGoogle, tok); err != nil {
		c.logger.Error("failed to set google cookie", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	c.logger.Info("Google authenticated")
	http.Redirect(w, r, "/?auth=ok", http.StatusSeeOther)
}
