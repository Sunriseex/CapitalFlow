package errors

import (
	"fmt"
	"strings"
)

type ErrorCode string

const (
	ErrValidation    ErrorCode = "VALIDATION_ERROR"
	ErrNotFound      ErrorCode = "NOT_FOUND"
	ErrStorage       ErrorCode = "STORAGE_ERROR"
	ErrCalculation   ErrorCode = "CALCULATION_ERROR"
	ErrConfiguration ErrorCode = "CONFIGURATION_ERROR"
	ErrSecurity      ErrorCode = "SECURITY_ERROR"
	ErrNotification  ErrorCode = "NOTIFICATION_ERROR"
	ErrBusinessLogic ErrorCode = "BUSINESS_LOGIC_ERROR"
)

type AppError struct {
	Code     ErrorCode
	Message  string
	Details  map[string]interface{}
	Original error
}

func (e *AppError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "[%s] %s", e.Code, e.Message)

	if len(e.Details) > 0 {
		sb.WriteString(" | Details: ")
		first := true
		for k, v := range e.Details {
			if !first {
				sb.WriteString(", ")
			}
			fmt.Fprintf(&sb, "%s=%v", k, v)
			first = false
		}
	}

	if e.Original != nil {
		fmt.Fprintf(&sb, " | Original: %v", e.Original)
	}

	return sb.String()
}

func NewValidationError(message string, details map[string]interface{}) *AppError {
	return &AppError{
		Code:    ErrValidation,
		Message: message,
		Details: details,
	}
}

func NewBusinessLogicError(message string, details map[string]interface{}) *AppError {
	return &AppError{
		Code:    ErrBusinessLogic,
		Message: message,
		Details: details,
	}
}

func NewCalculationError(message string, original error) *AppError {
	return &AppError{
		Code:     ErrCalculation,
		Message:  message,
		Original: original,
	}
}

func NewNotFoundError(resourceType, identifier string) *AppError {
	return &AppError{
		Code:    ErrNotFound,
		Message: fmt.Sprintf("%s not found", resourceType),
		Details: map[string]interface{}{
			"resource": resourceType,
			"id":       identifier,
		},
	}
}

func NewStorageError(operation string, original error) *AppError {
	return &AppError{
		Code:     ErrStorage,
		Message:  fmt.Sprintf("Storage operation failed: %s", operation),
		Details:  map[string]interface{}{"operation": operation},
		Original: original,
	}
}

func NewConfigurationError(message string, original error) *AppError {
	return &AppError{
		Code:     ErrConfiguration,
		Message:  message,
		Original: original,
	}
}

func WrapError(code ErrorCode, message string, original error) *AppError {
	return &AppError{
		Code:     code,
		Message:  message,
		Original: original,
	}
}

func IsValidationError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == ErrValidation
	}
	return false
}

func IsNotFoundError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == ErrNotFound
	}
	return false
}

func GetErrorCode(err error) ErrorCode {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return ""
}
