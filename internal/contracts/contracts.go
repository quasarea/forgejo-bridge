package contracts

import (
	"encoding/json"
	"fmt"
)

const SchemaVersion = "1.0"

type Warning struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type Page struct {
	Number int    `json:"number,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Total  int64  `json:"total,omitempty"`
	Next   string `json:"next,omitempty"`
}

type Metadata struct {
	ForgejoVersion string   `json:"forgejo_version,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
}

type ErrorDetail struct {
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	Retryable     bool           `json:"retryable"`
	Indeterminate bool           `json:"indeterminate"`
	HTTPStatus    int            `json:"http_status,omitempty"`
	Details       map[string]any `json:"details,omitempty"`
}

type Envelope struct {
	SchemaVersion string       `json:"schema_version"`
	OK            bool         `json:"ok"`
	Operation     string       `json:"operation"`
	RequestID     string       `json:"request_id"`
	Instance      string       `json:"instance,omitempty"`
	Data          any          `json:"data,omitempty"`
	Page          *Page        `json:"page,omitempty"`
	Warnings      []Warning    `json:"warnings"`
	Meta          Metadata     `json:"meta"`
	Error         *ErrorDetail `json:"error,omitempty"`
	Partial       bool         `json:"partial,omitempty"`
}

func Success(operation, requestID, instance string, data any) Envelope {
	return Envelope{
		SchemaVersion: SchemaVersion,
		OK:            true,
		Operation:     operation,
		RequestID:     requestID,
		Instance:      instance,
		Data:          data,
		Warnings:      []Warning{},
		Meta:          Metadata{},
	}
}

func Failure(operation, requestID, instance string, err *BridgeError) Envelope {
	return Envelope{
		SchemaVersion: SchemaVersion,
		OK:            false,
		Operation:     operation,
		RequestID:     requestID,
		Instance:      instance,
		Warnings:      []Warning{},
		Meta:          Metadata{},
		Error: &ErrorDetail{
			Code:          err.Code,
			Message:       err.Message,
			Retryable:     err.Retryable,
			Indeterminate: err.Indeterminate,
			HTTPStatus:    err.HTTPStatus,
			Details:       err.Details,
		},
	}
}

func Marshal(envelope Envelope) ([]byte, error) {
	return json.MarshalIndent(envelope, "", "  ")
}

type BridgeError struct {
	Code          string
	Message       string
	Retryable     bool
	Indeterminate bool
	HTTPStatus    int
	Details       map[string]any
	Cause         error
}

func (e *BridgeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *BridgeError) Unwrap() error { return e.Cause }

func NewError(code, message string) *BridgeError {
	return &BridgeError{Code: code, Message: message}
}

func WrapError(code, message string, cause error) *BridgeError {
	return &BridgeError{Code: code, Message: message, Cause: cause}
}
