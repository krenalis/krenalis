//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2015 Open2b
//

// Package capture allows to capture HTTP requests and responses.
package capture

import (
	"io"
	"net/http"
	"strconv"
	"sync"
)

var (
	crlf = []byte("\r\n")
)

type Flusher interface {
	Flush() error
}

// Request scrive la rappresentazione di r su out. Lo stato e le intestazioni
// sono copiate subito mentre il corpo è copiato solo durante la sua lettura.
// headers e body indicano se le instestazioni e il corpo devono essere copiati.
//
// Per copiare il corpo, Request sostituisce r.Body con un proprio Reader.
// Se out implementa Flusher al termine viene chiamato out.Flush.
func Request(r *http.Request, out io.Writer, headers, body bool) {
	if r.URL == nil {
		return
	}
	var method = r.Method
	if method == "" {
		method = "GET"
	}
	io.WriteString(out, method+" "+r.URL.RequestURI()+" "+r.Proto)
	if headers {
		out.Write(crlf)
		r.Header.Write(out)
	}
	if body && r.Body != nil {
		r.Body = &bodyCapture{out, r.Body, false, sync.Mutex{}}
	} else {
		if f, ok := out.(Flusher); ok {
			f.Flush()
		}
	}
}

// Response scrive la rappresentazione di r su out. Lo stato e le intestazioni
// sono copiate subito mentre il corpo è copiato solo durante la sua lettura.
// headers e body indicano se le instestazioni e il corpo devono essere copiati.
//
// Per copiare il corpo, Response sostituisce r.Body con un proprio Reader.
// Se out implementa Flusher al termine viene chiamato out.Flush.
func Response(r *http.Response, out io.Writer, headers, body bool) {
	io.WriteString(out, r.Status)
	if headers {
		out.Write(crlf)
		r.Header.Write(out)
	}
	if body && r.Body != nil {
		r.Body = &bodyCapture{out, r.Body, false, sync.Mutex{}}
	} else {
		if f, ok := out.(Flusher); ok {
			f.Flush()
		}
	}
}

type bodyCapture struct {
	out        io.Writer
	body       io.ReadCloser
	sentBody   bool
	sync.Mutex // used for the field 'out'.
}

func (c *bodyCapture) Close() error {
	c.Lock()
	defer c.Unlock()
	if c.out != nil {
		if f, ok := c.out.(Flusher); ok {
			f.Flush()
		}
		c.out = nil
	}
	return c.body.Close()
}

func (c *bodyCapture) Read(p []byte) (n int, err error) {
	n, err = c.body.Read(p)
	c.Lock()
	defer c.Unlock()
	if c.out != nil {
		if n > 0 {
			if !c.sentBody {
				c.out.Write(crlf)
				c.sentBody = true
			}
			if n == len(p) {
				c.out.Write(p)
			} else {
				c.out.Write(p[:n])
			}
		}
		if err != nil {
			if f, ok := c.out.(Flusher); ok {
				f.Flush()
			}
			c.out = nil
		}
	}
	return
}

type capturedResponse struct {
	w          http.ResponseWriter
	out        io.Writer
	statuses   []int
	header     http.Header
	body       bool
	sentHeader bool
	sentBody   bool
}

// ResponseWriter ritorna un http.ResponseWriter che chiama i metodi di w
// e scrive su out la rappresentazione della risposta. Se statuses non è
// vuoto, scrive solo se lo stato è uno di quelli elencati in statuses.
// headers e body indicano se le instestazioni e il corpo devono essere
// copiati.
func ResponseWriter(w http.ResponseWriter, out io.Writer, statuses []int, headers, body bool) http.ResponseWriter {
	var h http.Header
	if headers {
		h = make(http.Header)
	}
	return &capturedResponse{w: w, out: out, statuses: statuses, header: h, body: body}
}

// Header implementa http.ResponseWriter.
func (c *capturedResponse) Header() http.Header {
	if c.header != nil {
		return c.header
	}
	return c.w.Header()
}

// Write implementa http.ResponseWriter.
func (c *capturedResponse) Write(p []byte) (n int, err error) {
	if !c.sentHeader {
		c.WriteHeader(http.StatusOK)
	}
	n, err = c.w.Write(p)
	if c.out != nil && c.body && n > 0 {
		if !c.sentBody {
			c.out.Write(crlf)
			c.sentBody = true
		}
		if n == len(p) {
			c.out.Write(p)
		} else {
			c.out.Write(p[:n])
		}
	}
	return
}

// WriteHeader implementa http.ResponseWriter.
func (c *capturedResponse) WriteHeader(status int) {
	if c.header != nil {
		for k, vv := range c.header {
			for _, v := range vv {
				c.w.Header().Add(k, v)
			}
		}
	}
	c.w.WriteHeader(status)
	if c.mustCaptureStatus(status) {
		io.WriteString(c.out, strconv.Itoa(status)+" "+http.StatusText(status))
		if c.header != nil {
			c.out.Write(crlf)
			c.header.Write(c.out)
		}
	} else {
		c.out = nil
	}
	c.sentHeader = true
}

// Flush implementa http.Flusher.
func (c *capturedResponse) Flush() {
	if out, ok := c.out.(http.Flusher); ok {
		out.Flush()
	}
}

func (c *capturedResponse) mustCaptureStatus(status int) bool {
	if c.statuses == nil {
		return true
	}
	for _, s := range c.statuses {
		if s == status {
			return true
		}
	}
	return false
}

type roundTripper struct {
	tripper http.RoundTripper
	out     io.Writer
	headers bool
	body    bool
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	Request(req, rt.out, rt.headers, rt.body)
	res, err := rt.tripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	Response(res, rt.out, rt.headers, rt.body)
	return res, nil
}

// RoundTripper ritorna un http.RoundTripper che utilizza rt per eseguire la
// transazione HTTP e scrive su out la rappresentazione della richiesta e
// della risposta. headers e body indicano se le instestazioni e il corpo
// devono essere scritti su out.
func RoundTripper(rt http.RoundTripper, out io.Writer, headers, body bool) http.RoundTripper {
	return &roundTripper{rt, out, headers, body}
}
