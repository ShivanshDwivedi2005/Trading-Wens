package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strings"

	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/domain"
	"github.com/ShivanshDwivedi2005/Trading-Wens/backend/internal/platform/supabase"
)

const maxRequestSize = 64 << 10

type AuthService interface {
	Login(ctx context.Context, email, password string) (domain.AuthResult, error)
}

type AuthHandler struct {
	service AuthService
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed")
		return
	}

	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	if !validEmail(request.Email) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_email", "Enter a valid email address")
		return
	}
	if len(request.Password) < 8 {
		writeError(w, http.StatusUnprocessableEntity, "invalid_password", "Password must contain at least 8 characters")
		return
	}

	result, err := h.service.Login(r.Context(), request.Email, request.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return errors.New("Request body must be valid JSON with the expected fields")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("Request body must contain one JSON object")
	}
	return nil
}

func validEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}

func writeServiceError(w http.ResponseWriter, err error) {
	var apiErr *supabase.APIError
	if errors.As(err, &apiErr) {
		status := apiErr.Status
		if status < http.StatusBadRequest || status >= http.StatusInternalServerError {
			status = http.StatusBadGateway
		}
		writeError(w, status, apiErr.Code, apiErr.Message)
		return
	}
	writeError(w, http.StatusBadGateway, "authentication_unavailable", "Authentication service is unavailable")
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
