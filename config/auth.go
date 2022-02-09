package config

import (
	"encoding"
	"encoding/base64"
	"fmt"
	"net/url"
	"reflect"
)

// Base64 is a byte slice that encodes to and from base64-encoded strings.
type Base64 []byte

var (
	_ encoding.TextMarshaler   = (Base64)(nil)
	_ encoding.TextUnmarshaler = (*Base64)(nil)
)

// MarshalText implements encoding.TextMarshaler.
func (b Base64) MarshalText() ([]byte, error) {
	sz := base64.StdEncoding.EncodedLen(len(b))
	out := make([]byte, sz)
	base64.StdEncoding.Encode(out, b)
	return out, nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (b *Base64) UnmarshalText(in []byte) error {
	sz := base64.StdEncoding.DecodedLen(len(in))
	s := make([]byte, sz)
	n, err := base64.StdEncoding.Decode(s, in)
	if err != nil {
		return err
	}
	*b = s[:n]
	return nil
}

// Auth holds the specific configs for different authentication methods.
//
// These should be pointers to structs, so that it's possible to distinguish
// between "absent" and "present and misconfigured."
type Auth struct {
	PSK       *AuthPSK       `yaml:"psk,omitempty" json:"psk,omitempty"`
	Keyserver *AuthKeyserver `yaml:"keyserver,omitempty" json:"keyserver,omitempty"`
	MTLS      *AuthMTLS      `yaml:"mtls,omitempty" json:"mtls,omitempty"`
}

// Any reports whether any sort of authentication is configured.
func (a Auth) Any() bool {
	return len(a.populated()) != 0
}

// Populated returns a slice of elements that are not nil.
func (a *Auth) populated() (n []string) {
	v := reflect.ValueOf(a).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).IsNil() {
			n = append(n, t.Field(i).Name)
		}
	}
	return n
}

func (a *Auth) lint() ([]Warning, error) {
	ns := a.populated()
	if len(ns) > 1 {
		return []Warning{{
			msg: fmt.Sprintf(`multiple authentication methods defined: %v`, ns),
		}}, nil
	}
	return nil, nil
}

// AuthKeyserver is the configuration for doing authentication with the Quay
// keyserver protocol.
//
// The "Intraservice" key is only needed when the overall config mode is not
// "combo".
type AuthKeyserver struct {
	API          string `yaml:"api" json:"api"`
	Intraservice Base64 `yaml:"intraservice" json:"intraservice"`
}

func (a *AuthKeyserver) lint() ([]Warning, error) {
	return []Warning{{
		inner: fmt.Errorf(`authentication method deprecated: %w`, ErrDeprecated),
	}}, nil
}

// AuthPSK is the configuration for doing pre-shared key based authentication.
//
// The "Issuer" key is what the service expects to verify as the "issuer" claim.
type AuthPSK struct {
	Key    Base64   `yaml:"key" json:"key"`
	Issuer []string `yaml:"iss" json:"iss"`
}

func (a *AuthPSK) validate(_ Mode) ([]Warning, error) {
	if len(a.Key) == 0 {
		return nil, &Warning{
			msg: "key is empty",
		}
	}
	if len(a.Issuer) == 0 {
		return nil, &Warning{
			path: ".iss",
			msg:  "no issuers defined",
		}
	}
	return nil, nil
}

type AuthMTLS struct {
	Client struct {
		// The filesystem path where a TLS certificate can be read.
		//
		// It is an error for this to be the empty string.
		Cert string `yaml:"cert" json:"cert"`
		// The filesystem path where a TLS private key can be read.
		//
		// It is an error for this to be the empty string.
		Key string `yaml:"key" json:"key"`
	} `yaml:"client" json:"client"`
	Server struct {
		// RootCA ...
		//
		// If empty, the system trust store will be used.
		// It's possible this is a valid configuration, but unlikely.
		RootCA string `yaml:"root_ca" json:"root_ca"`
		// IDs is a list of URI SANs that are allowed.
		//
		// It is an error for this list to be empty or have invalid entries. See
		// also: https://datatracker.ietf.org/doc/html/rfc5280#section-4.2.1.6
		IDs []string `yaml:"ids" json:"ids"`
	} `yaml:"server" json:"server"`
}

func (a *AuthMTLS) validate(_ Mode) ([]Warning, error) {
	if len(a.Server.IDs) == 0 {
		return nil, &Warning{
			path: ".server.ids",
			msg:  "ids is empty",
		}
	}
	for i, id := range a.Server.IDs {
		u, err := url.Parse(id)
		if err != nil {
			return nil, err
		}
		if u.Scheme == "" {
			return nil, &Warning{
				path: fmt.Sprintf(".server.ids[%d]", i),
				msg:  "URI does not have a scheme",
			}
		}
	}
	if a.Client.Cert == "" {
		return nil, &Warning{
			path: ".client.cert",
			msg:  "no certificate provided",
		}
	}
	if a.Client.Key == "" {
		return nil, &Warning{
			path: ".client.key",
			msg:  "no key provided",
		}
	}
	return nil, nil
}
