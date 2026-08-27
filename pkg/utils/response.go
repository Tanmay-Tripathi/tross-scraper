// Package utils holds cross-cutting helpers. The response helpers here are the
// only sanctioned way for a controller to write an HTTP response: calling
// ctx.JSON directly bypasses the shared envelope and the error mapping.
package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/exceptions"
)

// Pagination describes the page of results carried in a list response.
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// ApiResponseV2 is the response envelope for all new endpoints.
type ApiResponseV2[T any] struct {
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Result     T           `json:"result,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// ApiResponseV1 is the legacy envelope, kept for endpoints whose contract
// predates V2. New endpoints must use SendApiResponseV2.
type ApiResponseV1 struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// errorResponse is the payload returned for any *ApplicationError.
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SendApiResponseV2 writes the standard envelope. When appErr is non-nil the
// error mapping wins and result/pagination are ignored.
func SendApiResponseV2[T any](ctx *gin.Context, result T, pagination *Pagination, appErr *exceptions.ApplicationError) {
	if appErr != nil {
		sendError(ctx, appErr)
		return
	}

	ctx.JSON(http.StatusOK, ApiResponseV2[T]{
		Code:       string(exceptions.ApiSuccessCode),
		Message:    exceptions.ApiSuccessMessage,
		Result:     result,
		Pagination: pagination,
	})
}

// SendApiResponseV1 writes the legacy envelope. Prefer SendApiResponseV2 unless
// you are deliberately matching an existing contract.
func SendApiResponseV1(ctx *gin.Context, data any, appErr *exceptions.ApplicationError) {
	if appErr != nil {
		sendError(ctx, appErr)
		return
	}

	ctx.JSON(http.StatusOK, ApiResponseV1{
		Code:    string(exceptions.ApiSuccessCode),
		Message: exceptions.ApiSuccessMessage,
		Data:    data,
	})
}

// sendError normalises a partially-filled ApplicationError before writing it,
// so a response can never go out with a blank code, message or status.
func sendError(ctx *gin.Context, appErr *exceptions.ApplicationError) {
	if appErr.ErrorCode == "" || appErr.ErrorMessage == "" {
		appErr = exceptions.New(exceptions.ErrorMessageNotFound)
	}

	status := appErr.HttpCode
	if status == 0 {
		status = http.StatusInternalServerError
	}

	ctx.AbortWithStatusJSON(status, errorResponse{
		Code:    string(appErr.ErrorCode),
		Message: appErr.ErrorMessage,
	})
}

// NewPagination computes the envelope's pagination block from the query that
// produced the page.
func NewPagination(page, pageSize int, totalItems int64) *Pagination {
	if pageSize <= 0 {
		pageSize = 1
	}

	totalPages := int(totalItems / int64(pageSize))
	if totalItems%int64(pageSize) != 0 {
		totalPages++
	}

	return &Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}
