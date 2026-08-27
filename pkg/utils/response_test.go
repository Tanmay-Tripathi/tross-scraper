package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/exceptions"
)

func init() { gin.SetMode(gin.TestMode) }

func newContext() (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return ctx, recorder
}

func TestSendApiResponseV2Success(t *testing.T) {
	ctx, recorder := newContext()

	SendApiResponseV2(ctx, map[string]string{"name": "tross"}, nil, nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["code"] != string(exceptions.ApiSuccessCode) {
		t.Errorf("code = %v, want %q", body["code"], exceptions.ApiSuccessCode)
	}
	if _, present := body["pagination"]; present {
		t.Error("pagination should be omitted when nil")
	}
}

func TestSendApiResponseV2Error(t *testing.T) {
	ctx, recorder := newContext()

	SendApiResponseV2[any](ctx, nil, nil, exceptions.New(exceptions.InvalidParams))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["code"] != string(exceptions.InvalidParams) {
		t.Errorf("code = %v, want %q", body["code"], exceptions.InvalidParams)
	}
	if !ctx.IsAborted() {
		t.Error("an error response must abort the handler chain")
	}
}

// A partially-filled error must never reach the client as a blank envelope.
func TestSendApiResponseV2NormalisesIncompleteError(t *testing.T) {
	ctx, recorder := newContext()

	SendApiResponseV2[any](ctx, nil, nil, &exceptions.ApplicationError{})

	if recorder.Code == 0 || recorder.Code == http.StatusOK {
		t.Fatalf("status = %d, want a non-OK status", recorder.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["code"] == "" || body["message"] == "" {
		t.Errorf("code and message must be filled in, got %#v", body)
	}
}

func TestNewPagination(t *testing.T) {
	tests := []struct {
		name           string
		page, pageSize int
		totalItems     int64
		wantTotalPages int
	}{
		{name: "exact multiple", page: 1, pageSize: 10, totalItems: 30, wantTotalPages: 3},
		{name: "partial last page", page: 1, pageSize: 10, totalItems: 31, wantTotalPages: 4},
		{name: "no items", page: 1, pageSize: 10, totalItems: 0, wantTotalPages: 0},
		{name: "fewer than one page", page: 1, pageSize: 10, totalItems: 4, wantTotalPages: 1},
		{name: "zero page size does not divide by zero", page: 1, pageSize: 0, totalItems: 5, wantTotalPages: 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NewPagination(tc.page, tc.pageSize, tc.totalItems)
			if got.TotalPages != tc.wantTotalPages {
				t.Errorf("TotalPages = %d, want %d", got.TotalPages, tc.wantTotalPages)
			}
			if got.TotalItems != tc.totalItems {
				t.Errorf("TotalItems = %d, want %d", got.TotalItems, tc.totalItems)
			}
		})
	}
}
