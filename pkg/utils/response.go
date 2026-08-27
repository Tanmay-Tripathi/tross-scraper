// Package utils holds cross-cutting helpers. These response helpers are the only
// sanctioned way to write a response; ctx.JSON bypasses the envelope.
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

// ApiResponseV2 is the response envelope every endpoint uses.
type ApiResponseV2[T any] struct {
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Result     T           `json:"result,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// errorResponse is the payload returned for any *ApplicationError.
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SendApiResponseV2 writes the standard envelope; a non-nil appErr wins.
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

// sendError normalises a partial error so no response goes out blank.
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

// NewPagination computes the pagination block for a page.
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
