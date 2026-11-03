package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func safeEqual(a, b string) bool {
	ha, hb := sha256.Sum256([]byte(a)), sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}

// Demo user store: "alice:alice123,bob:bob123". Swap for a real user table in production.
func parseUsers(s string) map[string]string {
	users := map[string]string{}
	for _, pair := range strings.Split(s, ",") {
		if u, p, ok := strings.Cut(strings.TrimSpace(pair), ":"); ok {
			users[u] = p
		}
	}
	return users
}

// cors answers preflight requests (which carry no token) before auth runs.
// CORS_ORIGIN: comma-separated list, or "*".
func cors(origins string, next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range strings.Split(origins, ",") {
		allowed[strings.TrimSpace(o)] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && (allowed["*"] || allowed[origin]) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,PUT,PATCH,POST,DELETE")
				if h := r.Header.Get("Access-Control-Request-Headers"); h != "" {
					w.Header().Set("Access-Control-Allow-Headers", h)
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// requireUser verifies the bearer JWT and passes the username to next.
func requireUser(next func(w http.ResponseWriter, r *http.Request, user string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scheme, tok, _ := strings.Cut(r.Header.Get("Authorization"), " ")
		if scheme != "Bearer" || tok == "" {
			writeErr(w, 401, "missing bearer token")
			return
		}
		claims := jwt.RegisteredClaims{}
		_, err := jwt.ParseWithClaims(tok, &claims, func(*jwt.Token) (any, error) { return jwtSecret, nil },
			jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || claims.Subject == "" {
			writeErr(w, 401, "invalid or expired token")
			return
		}
		next(w, r, claims.Subject)
	}
}

// proxy forwards to target keeping the full path. Internal services trust the
// gateway: the JWT is dropped and the verified user is passed as X-User.
func proxy(prefix, target string) http.HandlerFunc {
	u, err := url.Parse(target)
	if err != nil || target == "" {
		log.Fatalf("invalid upstream for %s: %q", prefix, target)
	}
	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(u)
			pr.Out.Host = u.Host
			pr.Out.Header.Del("Authorization")
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeErr(w, 502, "upstream "+prefix+" unavailable")
		},
	}
	return requireUser(func(w http.ResponseWriter, r *http.Request, user string) {
		r.Header.Set("X-User", user)
		rp.ServeHTTP(w, r)
	})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.RequestURI())
		next.ServeHTTP(w, r)
	})
}

func main() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is required")
	}
	jwtSecret = []byte(secret)
	expiresIn, err := time.ParseDuration(env("JWT_EXPIRES_IN", "1h"))
	if err != nil {
		log.Fatalf("invalid JWT_EXPIRES_IN: %v", err)
	}
	users := parseUsers(env("AUTH_USERS", "alice:alice123,bob:bob123"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })

	// Public: exchange credentials for a JWT.
	mux.HandleFunc("POST /auth/login", func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Username, Password string }
		json.NewDecoder(r.Body).Decode(&in)
		expected, ok := users[in.Username]
		if in.Username == "" || in.Password == "" || !ok || !safeEqual(in.Password, expected) {
			writeErr(w, 401, "invalid username or password")
			return
		}
		now := time.Now()
		tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			Subject:   in.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
		}).SignedString(jwtSecret)
		if err != nil {
			writeErr(w, 500, "could not sign token")
			return
		}
		writeJSON(w, 200, map[string]string{"token": tok, "username": in.Username})
	})

	// Everything below requires `Authorization: Bearer <jwt>`.
	mux.HandleFunc("GET /auth/me", requireUser(func(w http.ResponseWriter, _ *http.Request, user string) {
		writeJSON(w, 200, map[string]string{"username": user})
	}))

	// Routing table: path prefix -> upstream service.
	ticket := proxy("/tickets", os.Getenv("TICKET_SERVICE_URL"))
	booking := proxy("/bookings", os.Getenv("BOOKING_SERVICE_URL"))
	mux.Handle("/tickets", ticket)
	mux.Handle("/tickets/", ticket)
	mux.Handle("/bookings", booking)
	mux.Handle("/bookings/", booking)

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { writeErr(w, 404, "route not found") })

	port := env("PORT", "8080")
	log.Printf("gateway listening on %s", port)
	log.Fatal(http.ListenAndServe(":"+port,
		cors(env("CORS_ORIGIN", "http://localhost:5173,http://localhost:4173"), logRequests(mux))))
}
