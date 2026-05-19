package server

import (
	"errors"
	"net/http"

	"github.com/artmexbet/raibecas/services/gateway/internal/connector"
)

// mapConnectorError maps connector sentinel errors to HTTP status codes and error codes.
// It provides a unified error mapping for all gateway handlers.
func mapConnectorError(err error, fallbackMsg string) (status int, code string, message string) {
	if err == nil {
		return http.StatusOK, "", ""
	}

	switch {
	case errors.Is(err, connector.ErrInvalidRequest):
		return http.StatusBadRequest, "invalid_request", fallbackMsg
	case errors.Is(err, connector.ErrNotFound):
		return http.StatusNotFound, "not_found", fallbackMsg
	case errors.Is(err, connector.ErrUnauthorized):
		return http.StatusUnauthorized, "unauthorized", fallbackMsg
	case errors.Is(err, connector.ErrForbidden):
		return http.StatusForbidden, "forbidden", fallbackMsg
	default:
		return http.StatusInternalServerError, "internal_error", fallbackMsg
	}
}
