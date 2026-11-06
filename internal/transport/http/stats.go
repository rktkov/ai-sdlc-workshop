package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
	"github.com/rktkov/ytfeed/internal/transport/contract"
)

// StatsHandler serves the read-only stats projection.
func StatsHandler(service *feed.StatsService, userID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if service == nil {
			http.Error(w, "stats service is nil", http.StatusInternalServerError)
			return
		}
		days := 7
		if raw := r.URL.Query().Get("days"); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 1 || value > 90 {
				http.Error(w, "days must be between 1 and 90", http.StatusBadRequest)
				return
			}
			days = value
		}
		stats, err := service.Get(r.Context(), userID, days)
		if err != nil {
			http.Error(w, "stats unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(contract.StatsFromFeed(stats)); err != nil {
			http.Error(w, "encode stats", http.StatusInternalServerError)
		}
	}
}

type ratingRequest struct {
	Value    string   `json:"value"`
	BatchIDs []string `json:"batch_ids"`
}

// RatingHandler serves the weekly rating status and records one value.
func RatingHandler(service *feed.RatingService, userID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			http.Error(w, "rating service is nil", http.StatusInternalServerError)
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeRatingStatus(w, r, service, userID)
		case http.MethodPost:
			writeRating(w, r, service, userID)
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func writeRatingStatus(w http.ResponseWriter, r *http.Request, service *feed.RatingService, userID string) {
	prompt, err := service.ShouldPrompt(r.Context(), userID)
	if err != nil {
		http.Error(w, "rating unavailable", http.StatusInternalServerError)
		return
	}
	status := contract.RatingStatus{
		ShouldPrompt: prompt,
		WeekStart:    feed.WeekStart(time.Now().UTC()).Format("2006-01-02"),
	}
	last, err := service.Last(r.Context(), userID)
	if err == nil {
		value := contract.RatingFromFeed(last)
		status.Last = &value
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "encode rating", http.StatusInternalServerError)
	}
}

func writeRating(w http.ResponseWriter, r *http.Request, service *feed.RatingService, userID string) {
	var req ratingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	var (
		rating feed.WeeklyRating
		err    error
	)
	if req.Value == string(feed.RatingUnrated) {
		rating, err = service.Dismiss(r.Context(), userID, req.BatchIDs)
	} else {
		rating, err = service.Record(r.Context(), userID, feed.RatingValue(req.Value), req.BatchIDs)
	}
	if err != nil {
		http.Error(w, "invalid rating", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(contract.RatingFromFeed(rating)); err != nil {
		http.Error(w, "encode rating", http.StatusInternalServerError)
	}
}
