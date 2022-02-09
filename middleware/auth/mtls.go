package auth

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/quay/clair/config"
	"github.com/quay/zlog"
)

// MTLS implements the Checker interface.
type MTLS struct{}

// Check implements Checker.
//
// The instance used in the HTTP stack does not need to be a valid instance.
func (MTLS) Check(ctx context.Context, r *http.Request) bool {
	if r.TLS == nil {
		zlog.Error(ctx).Msg("no TLS connection state available; server is wired incorrectly")
		return false
	}
	if len(r.TLS.PeerCertificates) == 0 {
		zlog.Error(ctx).Msg("no peer certificates available; server is wired incorrectly")
		return false
	}
	return true
}

// MTLSClient holds the logic for client use.
type MTLSClient struct {
	Certificate *tls.Certificate
}

// From populates the receiver according to the passed-in configuration.
func (m *MTLSClient) From(cfg *config.AuthMTLS) error {
	c, err := tls.LoadX509KeyPair(cfg.Client.Cert, cfg.Client.Key)
	if err != nil {
		return err
	}
	m.Certificate = &c
	return nil
}

// Configure configures the passed tls.Config for client use.
func (m *MTLSClient) Configure(c *tls.Config) error {
	var err error
	c.GetClientCertificate, err = m.clientCert(c.GetClientCertificate)
	if err != nil {
		return err
	}
	return nil
}

type getCert func(*tls.CertificateRequestInfo) (*tls.Certificate, error)

// ClientCert returns the certificate in the MTLS's Key/CertFile members if the
// issuer is in the list of acceptable CAs sent by the server.
//
// If the passed function is not nil, it will be called if the issuer/CA check
// fails.
func (m *MTLSClient) clientCert(fn getCert) (getCert, error) {
	c := m.Certificate
	var err error
	c.Leaf, err = x509.ParseCertificate(c.Certificate[0])
	if err != nil {
		return nil, err
	}
	return func(csr *tls.CertificateRequestInfo) (*tls.Certificate, error) {
		for _, subj := range csr.AcceptableCAs {
			if bytes.Equal(subj, c.Leaf.RawIssuer) {
				return c, nil
			}
		}
		println("not offering client cert")
		if fn != nil {
			nc, err := fn(csr)
			if err != nil {
				return nil, err
			}
			return nc, nil
		}
		return nil, nil
	}, nil
}

// MTLSServer holds the logic for server use.
type MTLSServer struct {
	ClientCAs *x509.CertPool
	IDs       []*url.URL
}

// From populates the receiver according to the passed-in configuration.
func (m *MTLSServer) From(cfg *config.AuthMTLS) error {
	var err error
	if cfg.Server.RootCA != "" {
		m.ClientCAs = x509.NewCertPool()
		b, err := os.ReadFile(cfg.Server.RootCA)
		if err != nil {
			return err
		}
		if !m.ClientCAs.AppendCertsFromPEM(b) {
			return fmt.Errorf("no certificates loaded from %q", cfg.Server.RootCA)
		}
	} else {
		m.ClientCAs, err = x509.SystemCertPool()
		if err != nil {
			return err
		}
	}

	m.IDs = make([]*url.URL, len(cfg.Server.IDs))
	for i, id := range cfg.Server.IDs {
		m.IDs[i], err = url.Parse(id)
		if err != nil {
			return fmt.Errorf("bad URI %q: %w", id, err)
		}
	}

	return nil
}

// Configure configures the passed tls.Config for server use.
func (m *MTLSServer) Configure(c *tls.Config) error {
	c.ClientAuth = tls.RequireAndVerifyClientCert
	c.ClientCAs = m.ClientCAs
	var err error
	c.VerifyConnection, err = m.serverVerify(c.VerifyConnection)
	if err != nil {
		return err
	}
	return nil
}

type verifyConn func(tls.ConnectionState) error

// ServerVerify checks the verified peer certificate against the list of IDs in
// the MTLS.
//
// If the passed function is not nil, it will be called if the ID check fails.
func (m *MTLSServer) serverVerify(fn verifyConn) (verifyConn, error) {
	chk := make([][]byte, len(m.IDs))
	for i, u := range m.IDs {
		b, err := u.MarshalBinary()
		if err != nil {
			return nil, err
		}
		chk[i] = b
	}
	return func(s tls.ConnectionState) error {
		var ok bool
		// The first cert is the verified one, per the docs.
		c := s.PeerCertificates[0]
	ID:
		for _, want := range chk {
			for _, u := range c.URIs {
				got, err := u.MarshalBinary()
				if err != nil {
					// Highly unlikely this would happen -- it's already been
					// parsed by the server.
					return err
				}
				if bytes.Equal(want, got) {
					ok = true
					break ID
				}
			}
		}
		if ok {
			return nil
		}
		if fn != nil {
			return fn(s)
		}
		return errors.New("no certificates with allowed IDs found")
	}, nil
}
