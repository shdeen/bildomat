package httpapi

// Invariants tested:
//  1. Origin-scoped credentials: Given an HTTPS URL at the configured API origin, CredentialForURL
//     must return the exact supplied credential.
//  2. Credential selection under arbitrary URLs: For arbitrary URLs, any credential that
//     CredentialForURL returns must equal the supplied credential and belong to an absolute HTTPS
//     URL with the exact configured host.

import (
	neturl "net/url"
	"testing"
)

// TestCredFor verifies invariant #1: Origin-scoped credentials.
//
// What is being tested:
// Given an HTTPS URL at the configured API origin, CredentialForURL must return the exact supplied
// credential. Malformed URLs, relative URLs, different hosts, and changed schemes or ports must
// return an empty credential.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCredFor(t *testing.T) {
	const base = "https://api.prov.test/v1/videos"

	credential := HeaderCred("x-key", "s3cret")
	if got := CredentialForURL("https://api.prov.test/files/x.mp4", base, credential); got != credential {
		t.Errorf("✗ CredentialForURL(exact origin) = %v, want the credential returned", got)
	}

	denied := []struct{ name, url string }{
		{"different host", "https://cdn.prov.test/files/x.mp4"},
		{"deceptive suffix host", "https://api.prov.test.evil.example/x"},
		{"deceptive prefix host", "https://evilapi.prov.test/x"},
		{"scheme change", "http://api.prov.test/files/x.mp4"},
		{"port change", "https://api.prov.test:8443/files/x.mp4"},
		{"malformed", "://nope"},
		{"malformed brackets", "https://[::1"},
		{"relative path", "/files/x.mp4"},
		{"relative bare", "files/x.mp4"},
	}
	for _, c := range denied {
		if got := CredentialForURL(c.url, base, credential); got != (AuthCredential{}) {
			t.Errorf("✗ CredentialForURL(%s: %q) = %v, want no credential — the credential must not leave the origin", c.name, c.url, got)
		}
	}

	if !t.Failed() {
		t.Log("✓ CredentialForURL credentials only the exact API origin and yields no credential for every other URL")
	}
}

// FuzzCredFor verifies invariant #2: Credential selection under arbitrary URLs.
//
// What is being tested:
// For arbitrary URLs, any credential that CredentialForURL returns must equal the supplied
// credential and belong to an absolute HTTPS URL with the exact configured host.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzCredFor(f *testing.F) {
	const base = "https://api.prov.test/v1/videos"

	credential := HeaderCred("x-key", "s3cret")

	for _, s := range []string{
		"https://api.prov.test/files/x.mp4", "https://cdn.prov.test/x", "http://api.prov.test/x",
		"https://api.prov.test:8443/x", "://nope", "/relative", "https://[::1", "",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		got := CredentialForURL(raw, base, credential)
		if got != (AuthCredential{}) {
			u, err := neturl.Parse(raw)
			if err != nil || !u.IsAbs() || u.Scheme != "https" || u.Host != "api.prov.test" {
				t.Errorf("✗ CredentialForURL(%q) granted a credential to a nonmatching or invalid origin", raw)
			}

			if got != credential {
				t.Errorf("✗ CredentialForURL(%q) returned a different credential: %+v", raw, got)
			}
		}

		if !t.Failed() {
			t.Logf("✓ the credential decision held for the origin")
		}
	})
}
