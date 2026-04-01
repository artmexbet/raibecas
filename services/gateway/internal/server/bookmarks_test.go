package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	gatewayConnector "github.com/artmexbet/raibecas/services/gateway/internal/connector"
	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

func TestListBookmarksRequiresAuth(t *testing.T) {
	t.Parallel()

	connector := NewMockDocumentServiceConnector(t)
	srv := &Server{validator: validator.New(), documentConnector: connector}
	app := fiber.New()
	app.Get("/bookmarks", srv.listBookmarks)

	req := httptest.NewRequest(http.MethodGet, "/bookmarks", nil)
	resp, err := app.Test(req, int((5 * time.Second).Milliseconds()))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestCreateBookmarkRequiresAuth(t *testing.T) {
	t.Parallel()

	connector := NewMockDocumentServiceConnector(t)
	srv := &Server{validator: validator.New(), documentConnector: connector}
	app := fiber.New()
	app.Post("/bookmarks", srv.createBookmark)

	req := httptest.NewRequest(http.MethodPost, "/bookmarks", http.NoBody)
	req.Header.Set("Content-Type", fiber.MIMEApplicationJSON)
	resp, err := app.Test(req, int((5 * time.Second).Milliseconds()))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestMapBookmarkConnectorError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid request", err: gatewayConnector.ErrInvalidRequest, status: http.StatusBadRequest, code: "invalid_request"},
		{name: "not found", err: gatewayConnector.ErrNotFound, status: http.StatusNotFound, code: "not_found"},
		{name: "unauthorized", err: gatewayConnector.ErrUnauthorized, status: http.StatusUnauthorized, code: "unauthorized"},
		{name: "forbidden", err: gatewayConnector.ErrForbidden, status: http.StatusForbidden, code: "forbidden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status, code, _ := mapBookmarkConnectorError(tt.err, "fallback")
			if status != tt.status || code != tt.code {
				t.Fatalf("expected (%d, %s), got (%d, %s)", tt.status, tt.code, status, code)
			}
		})
	}
}

func TestCreateBookmarkMapsConnectorNotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	connector := NewMockDocumentServiceConnector(t)
	connector.EXPECT().CreateBookmark(mock.Anything, mock.MatchedBy(func(req domain.CreateBookmarkRequest) bool {
		return req.UserID == userID &&
			req.DocumentID == uuid.MustParse("22222222-2222-2222-2222-222222222222") &&
			req.Kind == domain.BookmarkKind("publication")
	})).Return(nil, gatewayConnector.ErrNotFound).Once()

	srv := &Server{
		validator:         validator.New(),
		documentConnector: connector,
	}
	app := fiber.New()
	app.Post("/bookmarks", func(c *fiber.Ctx) error {
		c.Locals(UserContextKey, &AuthUser{ID: userID, Role: "User"})
		return srv.createBookmark(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/bookmarks", strings.NewReader(`{"documentId":"22222222-2222-2222-2222-222222222222","kind":"publication"}`))
	req.Header.Set("Content-Type", fiber.MIMEApplicationJSON)
	resp, err := app.Test(req, int((5 * time.Second).Milliseconds()))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var errorResp domain.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if errorResp.Error != "not_found" {
		t.Fatalf("expected not_found error code, got %s", errorResp.Error)
	}
}
