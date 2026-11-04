package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/rktkov/ytfeed/internal/domain/taste"
)

type gestureRequest struct {
	GestureID  string            `json:"gesture_id"`
	CardID     string            `json:"card_id"`
	VideoID    string            `json:"video_id"`
	ChannelID  string            `json:"channel_id"`
	Topic      string            `json:"topic"`
	Gesture    string            `json:"gesture"`
	Reason     string            `json:"reason"`
	ScoredAxes []taste.AxisScore `json:"scored_axes"`
}

type gestureResponse struct {
	Applied   bool `json:"applied"`
	Duplicate bool `json:"duplicate"`
}

// GestureHandler records one gesture for the local owner.
func GestureHandler(service *taste.Service, userID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req gestureRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}

		result, err := service.Record(r.Context(), taste.Command{
			GestureID:  req.GestureID,
			UserID:     userID,
			CardID:     req.CardID,
			VideoID:    req.VideoID,
			ChannelID:  req.ChannelID,
			Topic:      req.Topic,
			Gesture:    taste.Gesture(req.Gesture),
			Reason:     taste.Reason(req.Reason),
			ScoredAxes: req.ScoredAxes,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(gestureResponse{Applied: result.Applied, Duplicate: result.Duplicate}); err != nil {
			http.Error(w, "encode gesture", http.StatusInternalServerError)
		}
	}
}
