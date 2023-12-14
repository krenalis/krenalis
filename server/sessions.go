// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2021 Open2b
//

package server

import (
	"net/http"
	"strings"
)

type sessionCookie struct {
	Member int
}

const (
	sessionCookieName = "api"
	sessionCookiePath = "/api/"
)

func (s *apisServer) removeSession(w http.ResponseWriter, r *http.Request) {
	session := s.getSession(r)
	if session == nil {
		return
	}

	// Remove the Go session.
	c := &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     sessionCookiePath,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	header := w.Header()
	if v := c.String(); v != "" {
		if cookies, ok := header["Set-Cookie"]; ok {
			prefix := sessionCookieName + "="
			for i, cookie := range cookies {
				if strings.HasPrefix(cookie, prefix) {
					cookies[i] = v + "; Priority=High"
					return
				}
			}
		}
		header.Add("Set-Cookie", v+"; Priority=High")
	}
	return
}

// getSession returns the session cookie from the request. If the request has no
// session cookie, returns nil.
func (s *apisServer) getSession(r *http.Request) *sessionCookie {
	cookie, _ := r.Cookie(sessionCookieName)
	if cookie == nil {
		return nil
	}
	sc := &sessionCookie{}
	err := s.secureCookie.Decode(sessionCookieName, cookie.Value, sc)
	if err != nil {
		return nil
	}
	return sc
}

// addSession creates the session and stores it inside the client.
func (s *apisServer) addSession(member int, w http.ResponseWriter, r *http.Request) error {
	err := s.storeSession(&sessionCookie{Member: member}, w)
	if err != nil {
		return err
	}
	return nil
}

// storeSession stores s in w.
func (s *apisServer) storeSession(sc *sessionCookie, w http.ResponseWriter) error {
	value, err := s.secureCookie.Encode(sessionCookieName, sc)
	if err != nil {
		return err
	}
	c := &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     sessionCookiePath,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	header := w.Header()
	if v := c.String(); v != "" {
		if cookies, ok := header["Set-Cookie"]; ok {
			prefix := sessionCookieName + "="
			for i, cookie := range cookies {
				if strings.HasPrefix(cookie, prefix) {
					cookies[i] = v + "; Priority=High"
					return nil
				}
			}
		}
		header.Add("Set-Cookie", v+"; Priority=High")
	}
	return nil
}
