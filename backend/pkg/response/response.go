package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse is the standard envelope for all API responses
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// APIError is the standard error object
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Meta carries optional pagination or informational metadata
type Meta struct {
	Total  int `json:"total,omitempty"`
	Page   int `json:"page,omitempty"`
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// OK sends a 200 OK response with data
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

// OKWithMeta sends a 200 OK response with data and pagination metadata
func OKWithMeta(c *gin.Context, data interface{}, meta Meta) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta:    &meta,
	})
}

// Created sends a 201 Created response
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    data,
	})
}

// BadRequest sends a 400 Bad Request response
func BadRequest(c *gin.Context, code, message string, details ...interface{}) {
	apiErr := &APIError{Code: code, Message: message}
	if len(details) > 0 {
		apiErr.Details = details[0]
	}
	c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Error:   apiErr,
	})
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(c *gin.Context, code, message string) {
	c.JSON(http.StatusUnauthorized, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

// Forbidden sends a 403 Forbidden response
func Forbidden(c *gin.Context, code, message string) {
	c.JSON(http.StatusForbidden, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

// NotFound sends a 404 Not Found response
func NotFound(c *gin.Context, code, message string) {
	c.JSON(http.StatusNotFound, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

// InternalServerError sends a 500 Internal Server Error response
func InternalServerError(c *gin.Context, code, message string) {
	c.JSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

// AbortBadRequest aborts the request with a 400 response
func AbortBadRequest(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

// AbortUnauthorized aborts the request with a 401 response
func AbortUnauthorized(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

// AbortForbidden aborts the request with a 403 response
func AbortForbidden(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusForbidden, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

// Conflict sends a 409 Conflict response
func Conflict(c *gin.Context, code, message string, details ...interface{}) {
	apiErr := &APIError{Code: code, Message: message}
	if len(details) > 0 {
		apiErr.Details = details[0]
	}
	c.JSON(http.StatusConflict, APIResponse{
		Success: false,
		Error:   apiErr,
	})
}

// AbortConflict aborts the request with a 409 Conflict response
func AbortConflict(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusConflict, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

// UnprocessableEntity sends a 422 Unprocessable Entity response
func UnprocessableEntity(c *gin.Context, code, message string, details ...interface{}) {
	apiErr := &APIError{Code: code, Message: message}
	if len(details) > 0 {
		apiErr.Details = details[0]
	}
	c.JSON(http.StatusUnprocessableEntity, APIResponse{
		Success: false,
		Error:   apiErr,
	})
}

// AbortInternalServerError aborts the request with a 500 response
func AbortInternalServerError(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

