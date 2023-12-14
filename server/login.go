//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"chichi/apis"
	"chichi/apis/errors"
)

// login logs a user in.
func (s *apisServer) login(w http.ResponseWriter, r *http.Request) {
	loginData := struct {
		Email    string
		Password string
	}{}
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&loginData)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	enc := json.NewEncoder(w)
	organization, err := s.apis.Organization(r.Context(), 1)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		slog.Error("cannot read organization", "err", err)
		return
	}
	memberID, err := organization.AuthenticateMember(r.Context(), loginData.Email, loginData.Password)
	if err != nil {
		if err, ok := err.(*errors.UnprocessableError); ok && err.Code == apis.AuthenticationFailed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			enc.Encode([]any{0, "AuthenticationFailed"})
			return
		}
		if err, ok := err.(errors.ResponseWriterTo); ok {
			_ = err.WriteTo(w)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		slog.Error("cannot log member", "err", err)
		return
	}

	err = s.addSession(organization.ID, w, r)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		slog.Error("cannot add session", "err", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = enc.Encode([]any{memberID, nil})
}

// logout logs the user out.
func (s *apisServer) logout(res http.ResponseWriter, req *http.Request) {
	s.removeSession(res, req)
}
