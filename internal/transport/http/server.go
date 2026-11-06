// Package httpapi serves the local HTTP API.
package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/rktkov/ytfeed/internal/transport/contract"
)

// Handler returns the local API handler.
func Handler(token, version string, schemaVersion int) http.Handler {
	return HandlerWithGestureRemote(token, version, schemaVersion, nil, false)
}

// HandlerWithRemote returns the API handler. A remote bind skips the loopback guard.
func HandlerWithRemote(token, version string, schemaVersion int, allowRemote bool) http.Handler {
	return HandlerWithGestureRemote(token, version, schemaVersion, nil, allowRemote)
}

// HandlerWithGesture adds the gesture endpoint.
func HandlerWithGesture(token, version string, schemaVersion int, gesture http.HandlerFunc) http.Handler {
	return HandlerWithGestureRemote(token, version, schemaVersion, gesture, false)
}

// HandlerWithGestureRemote adds the gesture endpoint and optional remote bind.
func HandlerWithGestureRemote(token, version string, schemaVersion int, gesture http.HandlerFunc, allowRemote bool) http.Handler {
	return HandlerWithFeatures(token, version, schemaVersion, gesture, nil, allowRemote)
}

// FeatureHandlers groups the optional local API endpoints.
type FeatureHandlers struct {
	Gesture http.HandlerFunc
	Feed    http.HandlerFunc
	Stats   http.HandlerFunc
	Rating  http.HandlerFunc
}

// HandlerWithFeatures adds the gesture and feed endpoints.
func HandlerWithFeatures(token, version string, schemaVersion int, gesture, feedHandler http.HandlerFunc, allowRemote bool) http.Handler {
	return HandlerWithHandlers(token, version, schemaVersion, FeatureHandlers{
		Gesture: gesture, Feed: feedHandler,
	}, allowRemote)
}

// HandlerWithHandlers adds every configured endpoint.
func HandlerWithHandlers(token, version string, schemaVersion int, handlers FeatureHandlers, allowRemote bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", health(version, schemaVersion))
	if handlers.Gesture != nil {
		mux.HandleFunc("/v1/gestures", handlers.Gesture)
	}
	if handlers.Feed != nil {
		mux.HandleFunc("/v1/feed", handlers.Feed)
	}
	if handlers.Stats != nil {
		mux.HandleFunc("/v1/stats", handlers.Stats)
	}
	if handlers.Rating != nil {
		mux.HandleFunc("/v1/rating", handlers.Rating)
	}
	if allowRemote {
		return auth(token, mux)
	}
	return loopback(auth(token, mux))
}

func health(version string, schemaVersion int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		payload := contract.Health{Status: "ok", Version: version, SchemaVersion: schemaVersion}
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			http.Error(w, "encode health", http.StatusInternalServerError)
		}
	}
}

func auth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		value, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(value), []byte(token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loopback(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
