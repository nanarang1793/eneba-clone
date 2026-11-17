package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"eneba/backend/internal/httpx"
)

// contextKey is a private type, so no other package can overwrite our value.
type contextKey struct{}

// RequireUser protects a route: it looks up the session cookie and either
// stores the user in the request context or answers 401.
//
//	r.With(authHandler.RequireUser).Get("/orders", listOrders)
//
// Handlers behind it read the user with CurrentUser(r).
func (h *Handler) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok, err := h.sessionUser(w, r)
		if err != nil {
			httpx.Internal(w, r, err)
			return
		}
		if !ok {
			httpx.Error(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "Please sign in")
			return
		}
		ctx := context.WithValue(r.Context(), contextKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// sessionUser finds the user for the request's session cookie.
// ok is false when there's no cookie, or it's unknown or expired; a dead
// cookie is also deleted from the browser.
func (h *Handler) sessionUser(w http.ResponseWriter, r *http.Request) (User, bool, error) {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return User{}, false, nil
	}
	user, err := h.store.UserBySession(r.Context(), c.Value)
	if errors.Is(err, ErrNoSession) {
		setSessionCookie(w, r, "", time.Unix(0, 0))
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	return user, true, nil
}

// CurrentUser returns the user stored by RequireUser.
// Only call it from handlers that sit behind RequireUser.
func CurrentUser(r *http.Request) User {
	u, _ := r.Context().Value(contextKey{}).(User)
	return u
}
