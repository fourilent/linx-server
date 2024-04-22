package main

import (
	"net/http"
)

const (
	cspHeader = "Content-Security-Policy"
	rpHeader  = "Referrer-Policy"
)

type CSP struct {
	h    http.Handler
	opts CSPOptions
}

type CSPOptions struct {
	policy         string
	referrerPolicy string
}

var defaultCSPOptions = CSPOptions{
	policy:         "default-src 'none'; img-src 'self'; media-src 'self'; style-src 'self' 'unsafe-inline'; frame-ancestors 'self';",
	referrerPolicy: "strict-origin",
}

var defaultFileCSPOptions = CSPOptions{
	policy:         "default-src 'none'; img-src 'self'; media-src 'self'; style-src 'self' 'unsafe-inline'; frame-ancestors 'self';",
	referrerPolicy: "strict-origin",
}

func (c CSP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// only add a CSP if one is not already set
	if w.Header().Get(cspHeader) == "" {
		w.Header().Add(cspHeader, c.opts.policy)
	}

	// only add a Referrer Policy if one is not already set
	if w.Header().Get(rpHeader) == "" {
		w.Header().Add(rpHeader, c.opts.referrerPolicy)
	}

	c.h.ServeHTTP(w, r)
}

func ContentSecurityPolicy(o CSPOptions) func(http.Handler) http.Handler {
	fn := func(h http.Handler) http.Handler {
		return CSP{h, o}
	}
	return fn
}
