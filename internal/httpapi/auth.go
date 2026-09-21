// Package httpapi sends authenticated HTTP requests and streams generated media.
package httpapi

import (
	"net/http"
	"net/url"
)

// AuthCredential contains an HTTP header credential.
//   - HeaderName: the request header that carries the credential
//   - Value: the complete credential header value
type AuthCredential struct{ HeaderName, Value string }

// Bearer returns an Authorization header credential containing an API key.
func Bearer(apiKey string) AuthCredential {
	return AuthCredential{HeaderName: headerAuthorization, Value: bearerScheme + " " + apiKey}
}

// HeaderCred returns a credential containing a header name and API key.
func HeaderCred(header, apiKey string) AuthCredential {
	return AuthCredential{HeaderName: header, Value: apiKey}
}

// set adds the credential to h when its header name is present.
func (c AuthCredential) set(h http.Header) {
	if c.HeaderName != "" {
		h.Set(c.HeaderName, c.Value)
	}
}

// CredentialForURL returns a credential when a candidate URL and API base have the same origin.
func CredentialForURL(candidateURL, apiBase string, credential AuthCredential) AuthCredential {
	parsedURL, err := url.Parse(candidateURL)
	if err != nil || !parsedURL.IsAbs() || parsedURL.Host == "" {
		return AuthCredential{}
	}

	baseURL, err := url.Parse(apiBase)
	if err != nil || parsedURL.Scheme != baseURL.Scheme || parsedURL.Host != baseURL.Host {
		return AuthCredential{}
	}

	return credential
}
