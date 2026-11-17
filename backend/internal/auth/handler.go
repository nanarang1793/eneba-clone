package auth

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"eneba/backend/internal/httpx"
)

const (
	cookieName  = "session"
	bcryptCost  = 10 // ~50–100 ms per hash: slow for attackers, fine for one login
	minPassword = 8
	maxPassword = 72 // bcrypt only uses the first 72 bytes
)

// dummyHash is compared against when the email doesn't exist, so a login for
// an unknown email takes as long as one with a wrong password. Otherwise the
// response time would reveal which emails have accounts.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcryptCost)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler { return &Handler{store: store} }

// Routes mounts /auth/register, /auth/login, /auth/logout and /auth/me.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/auth/register", h.register)
	r.Post("/auth/login", h.login)
	r.Post("/auth/logout", h.logout)
	r.Get("/auth/me", h.me)
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// POST /api/auth/register  {email, password, display_name}
// Creates the account and signs the user in right away (201 + cookie).
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	email, err := normalizeEmail(req.Email)
	if err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	if n := len(req.Password); n < minPassword || n > maxPassword {
		httpx.BadRequest(w, "Password must be 8 to 72 characters long")
		return
	}
	name := strings.TrimSpace(req.DisplayName)
	if name == "" {
		name = email[:strings.IndexByte(email, '@')] // "vykintas@…" -> "vykintas"
	}
	if utf8.RuneCountInString(name) > 40 {
		httpx.BadRequest(w, "Name must be at most 40 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	user, err := h.store.CreateUser(r.Context(), email, string(hash), name)
	if errors.Is(err, ErrEmailTaken) {
		httpx.Error(w, http.StatusConflict, "email_taken", "An account with this email already exists")
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if !h.startSession(w, r, user.ID) {
		return
	}
	httpx.JSON(w, http.StatusCreated, user)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// POST /api/auth/login  {email, password}
// Wrong email and wrong password give the same answer, so the endpoint
// can't be used to find out who has an account.
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, hash, found, err := h.store.UserByEmail(r.Context(), email)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if !found {
		hash = string(dummyHash) // same amount of work as a real check
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil || !found {
		httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "Wrong email or password")
		return
	}
	if !h.startSession(w, r, user.ID) {
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

// POST /api/auth/logout
// Deletes the session row and tells the browser to drop the cookie.
// Logging out when already logged out is fine (204 either way).
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		if err := h.store.DeleteSession(r.Context(), c.Value); err != nil {
			httpx.Internal(w, r, err)
			return
		}
	}
	setSessionCookie(w, r, "", time.Unix(0, 0)) // an expiry in the past deletes the cookie
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/auth/me -> {"user": {...}} or {"user": null}
// Being signed out is a normal state, not an error, so this answers 200
// either way. (A 401 here would show up as a red error in the browser
// console on every page a signed-out visitor opens.)
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, ok, err := h.sessionUser(w, r)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if !ok {
		httpx.JSON(w, http.StatusOK, map[string]any{"user": nil})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"user": user})
}

// startSession creates a session and sets the cookie. On failure it has
// already written a 500 and returns false.
func (h *Handler) startSession(w http.ResponseWriter, r *http.Request, userID string) bool {
	token, expires, err := h.store.CreateSession(r.Context(), userID)
	if err != nil {
		httpx.Internal(w, r, err)
		return false
	}
	setSessionCookie(w, r, token, expires)
	return true
}

// setSessionCookie writes the session cookie:
//   - HttpOnly: page JavaScript can't read it, so an XSS bug can't steal it.
//   - SameSite=Lax: the browser won't attach it to POSTs coming from other sites.
//   - Secure: only sent over HTTPS (in production; local dev runs on http).
func setSessionCookie(w http.ResponseWriter, r *http.Request, value string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
	})
}

// normalizeEmail trims and lowercases, and checks it looks like an email.
func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > 254 || !strings.Contains(email[strings.IndexByte(email, '@'):], ".") {
		return "", errors.New("Please enter a valid email address")
	}
	return email, nil
}
