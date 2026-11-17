package auth

// Integration test of the whole sign-in flow through real HTTP requests
// (TEST_DATABASE_URL; skipped without it).

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"eneba/backend/internal/testdb"
)

func TestSignInFlow(t *testing.T) {
	pool := testdb.Open(t)
	h := NewHandler(NewStore(pool))
	r := chi.NewRouter()
	h.Routes(r)
	r.With(h.RequireUser).Get("/private", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello " + CurrentUser(r).Email))
	})

	email := "flow-" + testdb.Tag() + "@test.example"
	t.Cleanup(func() { testdb.Exec(t, pool, `DELETE FROM users WHERE email = $1`, email) })

	// do sends a request (with an optional session cookie) and returns the response.
	do := func(method, path, body, cookie string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: cookieName, Value: cookie})
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}
	sessionCookie := func(rec *httptest.ResponseRecorder) *http.Cookie {
		for _, c := range rec.Result().Cookies() {
			if c.Name == cookieName {
				return c
			}
		}
		t.Fatal("no session cookie set")
		return nil
	}
	meEmail := func(cookie string) string {
		var body struct {
			User *User `json:"user"`
		}
		_ = json.Unmarshal(do("GET", "/auth/me", "", cookie).Body.Bytes(), &body)
		if body.User == nil {
			return ""
		}
		return body.User.Email
	}

	// Register signs you in, with an HttpOnly cookie.
	rec := do("POST", "/auth/register", `{"email":"`+strings.ToUpper(email)+`","password":"password123"}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body)
	}
	c := sessionCookie(rec)
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode {
		t.Errorf("cookie must be HttpOnly and SameSite=Lax: %+v", c)
	}
	if got := meEmail(c.Value); got != email {
		t.Errorf("me after register: want %s, got %q", email, got)
	}

	// The database stores a bcrypt hash and the token's SHA-256, never the secrets.
	var hash string
	var tokens int
	ctx := context.Background()
	sum := sha256.Sum256([]byte(c.Value))
	_ = pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE email = $1`, email).Scan(&hash)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE token_hash = $1`, sum[:]).Scan(&tokens)
	if !strings.HasPrefix(hash, "$2a$") || tokens != 1 {
		t.Errorf("want bcrypt hash and one hashed session, got hash=%.7s… sessions=%d", hash, tokens)
	}

	// Same email in other letters: 409.
	if rec := do("POST", "/auth/register", `{"email":"`+email+`","password":"password123"}`, ""); rec.Code != http.StatusConflict {
		t.Errorf("duplicate register: want 409, got %d", rec.Code)
	}

	// Protected route: 401 without a cookie, 200 with one.
	if rec := do("GET", "/private", "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("private without cookie: want 401, got %d", rec.Code)
	}
	if rec := do("GET", "/private", "", c.Value); rec.Code != http.StatusOK {
		t.Errorf("private with cookie: want 200, got %d", rec.Code)
	}

	// Logout kills the session: the old cookie stops working.
	if rec := do("POST", "/auth/logout", "", c.Value); rec.Code != http.StatusNoContent {
		t.Errorf("logout: %d", rec.Code)
	}
	if got := meEmail(c.Value); got != "" {
		t.Errorf("old cookie still signed in as %s", got)
	}
	if rec := do("GET", "/private", "", c.Value); rec.Code != http.StatusUnauthorized {
		t.Errorf("private after logout: want 401, got %d", rec.Code)
	}

	// Wrong password and unknown email give the same answer.
	wrong := do("POST", "/auth/login", `{"email":"`+email+`","password":"nope-nope"}`, "")
	unknown := do("POST", "/auth/login", `{"email":"nobody-`+email+`","password":"nope-nope"}`, "")
	if wrong.Code != 401 || unknown.Code != 401 || wrong.Body.String() != unknown.Body.String() {
		t.Errorf("wrong password vs unknown email must look identical: %d %s / %d %s",
			wrong.Code, wrong.Body, unknown.Code, unknown.Body)
	}

	// Correct login works again.
	rec = do("POST", "/auth/login", `{"email":"`+email+`","password":"password123"}`, "")
	if rec.Code != http.StatusOK || meEmail(sessionCookie(rec).Value) != email {
		t.Errorf("login: %d %s", rec.Code, rec.Body)
	}

	// Form posts (what a cross-site HTML form sends) are refused.
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader("email=x&password=y"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("form-encoded login: want 400, got %d", rec.Code)
	}
}
