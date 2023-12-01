// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2021 Open2b
//

package admin

import (
	"net/http"
	"strings"
)

type sessionCookie struct {
	Member int
}

const sessionCookieName = "admin"

func (admin *admin) removeSession(w http.ResponseWriter, r *http.Request) error {
	session := admin.getSession(r)
	if session == nil {
		return nil
	}

	// Remove the Go session.
	c := &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/admin",
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
					return nil
				}
			}
		}
		header.Add("Set-Cookie", v+"; Priority=High")
	}
	return nil
}

// getSession returns the session cookie from the request. If the request has no
// session cookie, returns nil.
func (admin *admin) getSession(r *http.Request) *sessionCookie {
	cookie, _ := r.Cookie(sessionCookieName)
	if cookie == nil {
		return nil
	}
	s := &sessionCookie{}
	err := admin.secureCookie.Decode(sessionCookieName, cookie.Value, s)
	if err != nil {
		return nil
	}
	return s
}

// addSession creates the session and stores it inside the client.
func (admin *admin) addSession(member int, w http.ResponseWriter, r *http.Request) error {
	err := admin.storeSession(&sessionCookie{Member: member}, w)
	if err != nil {
		return err
	}
	return nil
}

// storeSession stores s in w.
func (admin *admin) storeSession(s *sessionCookie, w http.ResponseWriter) error {
	value, err := admin.secureCookie.Encode(sessionCookieName, s)
	if err != nil {
		return err
	}
	c := &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/admin",
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
