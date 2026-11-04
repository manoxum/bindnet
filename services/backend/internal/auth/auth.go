// Package auth guarda o administrador do painel e a protecao de sessao
// (cookie httpOnly assinado com HMAC-SHA256, sem dependencia externa). Os
// pacotes de rota usam RequireSession/SessionUser/Administrator.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const (
	sessionCookieName = "bindnet_session"
	// Prazo compativel com o limite de 400 dias do Chrome; cada visita renova o prazo.
	sessionCookieLifetime = 400 * 24 * time.Hour
)

type session struct {
	Username string `json:"username"`
	Exp      int64  `json:"exp,omitempty"` // somente tokens legados tem expiracao
}

// issueToken cria um token assinado (payload base64url + assinatura
// HMAC-SHA256) - um substituto minimalista de JWT sem depender de
// bibliotecas externas, guardado num cookie httpOnly.
func issueToken(username string, secret []byte) (string, error) {
	s := session{Username: username}
	payload, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	return payloadB64 + "." + sign(payloadB64, secret), nil
}

func validateToken(token string, secret []byte) (string, bool) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", false
	}
	if !hmac.Equal([]byte(sign(parts[0], secret)), []byte(parts[1])) {
		return "", false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	var s session
	if err := json.Unmarshal(payload, &s); err != nil {
		return "", false
	}
	if s.Username == "" || (s.Exp != 0 && time.Now().Unix() >= s.Exp) {
		return "", false
	}
	return s.Username, true
}

func sign(payloadB64 string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payloadB64))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type auditRecorder interface {
	Record(context.Context, string, string, map[string]any)
}

func RegisterRoutes(mux *http.ServeMux, admin *Administrator, aud auditRecorder) {
	mux.HandleFunc("POST /api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var c credentials
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			http.Error(w, "corpo invalido", http.StatusBadRequest)
			return
		}
		if c.Username != admin.Username || !admin.validPassword(c.Password) {
			http.Error(w, "usuario ou senha invalidos", http.StatusUnauthorized)
			return
		}
		token, err := issueToken(admin.Username, admin.secret())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		setSessionCookie(w, token)
		aud.Record(r.Context(), "login", admin.Username, nil)
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		username, _ := SessionUser(r, admin)
		http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1})
		aud.Record(r.Context(), "logout", username, nil)
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		username, ok := SessionUser(r, admin)
		if !ok {
			http.Error(w, "nao autenticado", http.StatusUnauthorized)
			return
		}
		// Renova o armazenamento no navegador e migra sessoes legadas ainda validas.
		token, err := issueToken(username, admin.secret())
		if err != nil {
			http.Error(w, "falha ao renovar sessao", http.StatusInternalServerError)
			return
		}
		setSessionCookie(w, token)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]string{"username": username})
	})
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionCookieLifetime.Seconds()),
		Expires:  time.Now().Add(sessionCookieLifetime),
	})
}

func SessionUser(r *http.Request, admin *Administrator) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}
	return validateToken(cookie.Value, admin.secret())
}

// RequireSession protege as rotas de negocio - so libera a requisicao se o
// cookie de sessao emitido em /api/auth/login for valido.
func RequireSession(admin *Administrator, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := SessionUser(r, admin); !ok {
			http.Error(w, "nao autenticado", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
