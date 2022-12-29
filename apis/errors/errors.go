//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package errors is a drop-in replacement for the errors package from the Go
// standard library and provides additional error types to handle errors that
// may occur while serving HTTP requests and need to be returned to the client.
//
// It includes functions for creating errors with specific HTTP response status
// codes, such as BadRequest, NotFound, and Unprocessable. The returned errors
// implement the ResponseWriterTo interface, which allows them to be written to
// a http.ResponseWriter value and sent to the client as the HTTP response.
//
// The BadRequest function can be used when an invalid HTTP request is received
// from the client. For example, it could be used if a required field in the
// request is missing or if the provided data is invalid.
//
// The NotFound function can be used when a request is made for a resource that
// does not exist or is not available. For example, it could be used if someone
// tries to access a connection that does not exist or when trying to access a
// workspace to which permission to access is not granted.
//
// The Unprocessable function should be used when the request cannot be
// satisfied not due to formal errors with the arguments, but due to argument
// values that are not compliant with the current state. For example, this
// function could be used if a request is made to update a resource, but the
// provided data is not valid given the current state of the resource.
package errors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// Code represents an error code in an unprocessable error.
type Code string

// The ResponseWriterTo interface is implemented by errors that can be written
// to a http.ResponseWriter value. A web server can call WriteTo to send the
// error to the client.
type ResponseWriterTo interface {
	WriteTo(w http.ResponseWriter) error
}

// BadRequest returns an error that formats as the given text, and its WriteTo
// method replies to the request with an HTTP 400 bad request error.
//
// If format includes a %w verb with an error operand, BadRequest returns an
// error that implements an Unwrap method returning the operand, and the
// WriteTo method reports the error message in the "details" key.
//
// It can be used when an invalid call is received. For example, it could be
// used if a required argument is empty or if the provided data is not formally
// valid.
func BadRequest(format string, a ...any) error {
	e := fmt.Errorf(format, a...)
	return &badRequestError{s: e.Error(), err: Unwrap(e)}
}

// badRequestError is an implementation of error used to represent a bad
// request error.
type badRequestError struct {
	s   string
	err error
}

// Error implements the errors interface.
func (e *badRequestError) Error() string {
	return e.s
}

// Unwrap returns the wrapped error.
func (e *badRequestError) Unwrap() error {
	return e.err
}

// WriteTo implements the ResponseWriterTo interface.
func (e *badRequestError) WriteTo(w http.ResponseWriter) error {
	var details string
	if e.err != nil {
		details = e.err.Error()
	}
	return writeTo(w, http.StatusBadRequest, "BadRequest", e.s, details)
}

// NotFound returns an error that formats as the given text, and its WriteTo
// method replies to the request with an HTTP 404 not found error.
//
// It can be used when a call is made for an entity that does not exist. For
// example, it could be used when trying to access a connection that does not
// exist or when trying to access a workspace to which access is not granted.
func NotFound(format string, a ...any) *NotFoundError {
	return &NotFoundError{fmt.Sprintf(format, a...)}
}

// NotFoundError is an implementation of error used to represent a not found
// error.
type NotFoundError struct {
	Message string
}

// Error implements the errors interface.
func (e *NotFoundError) Error() string {
	return e.Message
}

// WriteTo implements the ResponseWriterTo interface.
func (e *NotFoundError) WriteTo(w http.ResponseWriter) error {
	return writeTo(w, http.StatusNotFound, "NotFound", e.Message, "")
}

// Unprocessable returns an error with the given code that formats as the given
// text, and its WriteTo method replies to the request with an HTTP 422
// unprocessable entity error.
//
// If format includes a %w verb with an error operand, Unprocessable returns
// an error that implements an Unwrap method returning the operand, and the
// WriteTo method reports the error message in the "details" key.
//
// Unprocessable function should be used when the request cannot be satisfied
// not due to formal errors with the arguments, but due to argument values that
// are not compliant with the current state. For example, it could be used if a
// request is made to update a connection, but the provided data is not valid
// given its current state.
func Unprocessable(code Code, format string, a ...any) *UnprocessableError {
	e := fmt.Errorf(format, a...)
	return &UnprocessableError{Code: code, Message: e.Error(), Err: Unwrap(e)}
}

// An UnprocessableError value is returned when an argument is unprocessable.
type UnprocessableError struct {
	Code    Code
	Message string
	Err     error
}

// Error implements the errors interface.
func (e *UnprocessableError) Error() string {
	return e.Message
}

// Unwrap returns the wrapped error.
func (e *UnprocessableError) Unwrap() error {
	return e.Err
}

// WriteTo implements the ResponseWriterTo interface.
func (e *UnprocessableError) WriteTo(w http.ResponseWriter) error {
	var details string
	if e.Err != nil {
		details = e.Err.Error()
	}
	return writeTo(w, http.StatusUnprocessableEntity, e.Code, e.Message, details)
}

// marshalString marshals s as a JSON string and returns the result.
func marshalString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// writeTo writes code, message and, details as JSON to w with status as HTTP
// response status code. It returns an error from the json package if an error
// occurs marshalling data and another error if an error occurs writing to w.
func writeTo(w http.ResponseWriter, status int, code Code, message, details string) error {
	var b bytes.Buffer
	b.WriteString(`{"error":{"code":`)
	b.WriteString(marshalString(string(code)))
	b.WriteString(`,"message":`)
	b.WriteString(marshalString(message))
	if details != "" {
		b.WriteString(`,"details":`)
		b.WriteString(marshalString(message))
	}
	b.WriteString(`}}`)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(b.Len()))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, err := w.Write(b.Bytes())
	return err
}
