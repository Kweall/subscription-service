package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"subscription-service/internal/repository"
	"subscription-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	svc service.SubscriptionService
}

func NewHandler(svc service.SubscriptionService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) OpenAPIDoc(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "docs/openapi.yaml")
}

func (h *Handler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	var in createReq
	if err := decodeJSON(r.Body, &in); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if strings.TrimSpace(in.ServiceName) == "" || in.Price < 0 {
		respondErr(w, http.StatusBadRequest, "service_name required, price must be >= 0")
		return
	}
	if _, err := uuid.Parse(in.UserID); err != nil {
		respondErr(w, http.StatusBadRequest, "user_id must be uuid")
		return
	}
	startDate, err := parseMonthYear(in.StartDate)
	if err != nil {
		respondErr(w, http.StatusBadRequest, "start_date must be MM-YYYY")
		return
	}
	var endDatePtr *time.Time
	if in.EndDate != nil {
		ed, err := parseMonthYear(*in.EndDate)
		if err != nil {
			respondErr(w, http.StatusBadRequest, "end_date must be MM-YYYY")
			return
		}
		endDatePtr = &ed
		if endDatePtr.Before(startDate) {
			respondErr(w, http.StatusBadRequest, "end_date must be >= start_date")
			return
		}
	}

	created, err := h.svc.CreateSubscription(r.Context(), service.CreateInput{
		ServiceName: in.ServiceName,
		Price:       in.Price,
		UserID:      in.UserID,
		StartDate:   startDate,
		EndDate:     endDatePtr,
	})
	if err != nil {
		log.Error().Err(err).Msg("CreateSubscription failed")
		respondErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/subscriptions/"+created.ID)

	log.Info().
		Msgf("A subscription to %s was added for user %s from %s to %s for %v units",
			created.ServiceName, created.UserID, created.StartDate.Format("2006-01-02"), created.EndDate.Format("2006-01-02"), created.Price)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *Handler) GetSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		respondErr(w, http.StatusBadRequest, "id must be uuid")
		return
	}
	s, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			respondErr(w, http.StatusNotFound, "not found")
			return
		}
		respondErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var filter repository.ListFilter
	if u := q.Get("user_id"); u != "" {
		filter.UserID = &u
	}
	if s := q.Get("service_name"); s != "" {
		filter.ServiceName = &s
	}
	limit := 50
	if l := q.Get("limit"); l != "" {
		if vi, err := strconv.Atoi(l); err == nil && vi > 0 && vi <= 1000 {
			limit = vi
		}
	}
	offset := 0
	if o := q.Get("offset"); o != "" {
		if vi, err := strconv.Atoi(o); err == nil && vi >= 0 {
			offset = vi
		}
	}
	filter.Limit = limit
	filter.Offset = offset

	subs, err := h.svc.ListSubscriptions(r.Context(), filter)
	if err != nil {
		respondErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

func (h *Handler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		respondErr(w, http.StatusBadRequest, "id must be uuid")
		return
	}
	var in createReq
	if err := decodeJSON(r.Body, &in); err != nil {
		respondErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if strings.TrimSpace(in.ServiceName) == "" || in.Price < 0 {
		respondErr(w, http.StatusBadRequest, "service_name required, price >= 0")
		return
	}
	if _, err := uuid.Parse(in.UserID); err != nil {
		respondErr(w, http.StatusBadRequest, "user_id must be uuid")
		return
	}
	startDate, err := parseMonthYear(in.StartDate)
	if err != nil {
		respondErr(w, http.StatusBadRequest, "start_date must be MM-YYYY")
		return
	}
	var endDatePtr *time.Time
	if in.EndDate != nil {
		ed, err := parseMonthYear(*in.EndDate)
		if err != nil {
			respondErr(w, http.StatusBadRequest, "end_date must be MM-YYYY")
			return
		}
		endDatePtr = &ed
		if endDatePtr.Before(startDate) {
			respondErr(w, http.StatusBadRequest, "end_date must be >= start_date")
			return
		}
	}

	updated, err := h.svc.UpdateSubscription(r.Context(), id, service.UpdateInput{
		ServiceName: in.ServiceName,
		Price:       in.Price,
		UserID:      in.UserID,
		StartDate:   startDate,
		EndDate:     endDatePtr,
	})
	if err != nil {
		if err == repository.ErrNotFound {
			respondErr(w, http.StatusNotFound, "not found")
			return
		}
		respondErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info().
		Msgf("The subscription for user %s was updated: %s for %v units", updated.UserID, updated.ServiceName, updated.Price)
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		respondErr(w, http.StatusBadRequest, "id must be uuid")
		return
	}

	sub, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			respondErr(w, http.StatusNotFound, "not found")
			return
		}
		respondErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.svc.DeleteSubscription(r.Context(), id); err != nil {
		if err == repository.ErrNotFound {
			respondErr(w, http.StatusNotFound, "not found")
			return
		}
		respondErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info().
		Msgf("The subscription for user %s was unsubscribed", sub.UserID)

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetTotalCost(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		http.Error(w, "`from` and `to` required", http.StatusBadRequest)
		return
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		http.Error(w, "invalid from", http.StatusBadRequest)
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		http.Error(w, "invalid to", http.StatusBadRequest)
		return
	}
	if from.After(to) {
		respondErr(w, http.StatusBadRequest, "invalid date range: 'from' must be before 'to'")
		return
	}

	var uidPtr, snPtr *string
	if uid := r.URL.Query().Get("user_id"); uid != "" {
		uidPtr = &uid
	}
	if sn := r.URL.Query().Get("service_name"); sn != "" {
		snPtr = &sn
	}

	total, err := h.svc.SumForPeriod(r.Context(), from, to, uidPtr, snPtr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"total": total})
}

type createReq struct {
	ServiceName string  `json:"service_name"`
	Price       int     `json:"price"`
	UserID      string  `json:"user_id"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
}

func parseMonthYear(s string) (time.Time, error) {
	t, err := time.Parse("01-2006", s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func decodeJSON(r io.ReadCloser, v interface{}) error {
	defer r.Close()
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func respondErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
