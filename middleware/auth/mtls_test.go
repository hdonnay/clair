package auth

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/quay/zlog"
)

type mtlsTestcase struct {
	CA     *x509.Certificate
	Client tls.Certificate
	ID     string
	Nonce  string
}

func (tc *mtlsTestcase) Setup(t *testing.T) {
	tc.ID = `test://mtls/` + t.Name()
	caIn := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"mtls_test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * time.Minute),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	// This isn't good key material, don't do this for real.
	caPriv, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	caPub := &caPriv.PublicKey
	caDer, err := x509.CreateCertificate(crand.Reader, &caIn, &caIn, caPub, caPriv)
	if err != nil {
		t.Fatal(err)
	}
	tc.CA, err = x509.ParseCertificate(caDer)
	if err != nil {
		t.Fatal(err)
	}

	id, err := url.Parse(tc.ID)
	if err != nil {
		t.Fatal(err)
	}
	clientIn := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"mtls_test"},
		},
		URIs:                  []*url.URL{id},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * time.Minute),
		IsCA:                  false,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	clientPriv, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	clientDer, err := x509.CreateCertificate(crand.Reader, &clientIn, tc.CA, &clientPriv.PublicKey, caPriv)
	if err != nil {
		t.Fatal(err)
	}
	tc.Client.PrivateKey = clientPriv
	tc.Client.Certificate = append(tc.Client.Certificate, clientDer)
	tc.Client.Leaf, err = x509.ParseCertificate(clientDer)
	if err != nil {
		t.Fatal(err)
	}

	b := make([]byte, 16)
	if _, err := io.ReadFull(crand.Reader, b); err != nil {
		t.Fatal(err)
	}
	tc.Nonce = base64.StdEncoding.EncodeToString(b)
}

func (tc *mtlsTestcase) Handler(t *testing.T) http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, tc.Nonce)
	})
	return Handler(h, MTLS{})
}

func (tc *mtlsTestcase) Run(ctx context.Context) func(*testing.T) {
	return func(t *testing.T) {
		tc.Setup(t)
		// t.Log(tc)
		mc := MTLSClient{
			Certificate: &tc.Client,
		}
		ms := MTLSServer{
			ClientCAs: x509.NewCertPool(),
		}
		u, err := url.Parse(tc.ID)
		if err != nil {
			t.Fatal(err)
		}
		ms.IDs = append(ms.IDs, u)
		ms.ClientCAs.AddCert(tc.CA)

		// Set up the http server.
		h := tc.Handler(t)
		if t.Failed() {
			return
		}
		srv := httptest.NewUnstartedServer(h)
		srv.Config.BaseContext = func(_ net.Listener) context.Context { return ctx }
		srv.TLS = &tls.Config{}
		if err := ms.Configure(srv.TLS); err != nil {
			t.Error(err)
		}
		srv.StartTLS()
		c := srv.Client()
		if err := mc.Configure(c.Transport.(*http.Transport).TLSClientConfig); err != nil {
			t.Error(err)
		}
		defer srv.Close()

		// Mint a request.
		req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
		if err != nil {
			t.Error(err)
		}

		// Execute the request and read back the body.
		res, err := c.Do(req)
		if err != nil {
			t.Error(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Error(fmt.Errorf("unexpected response: %d %s", res.StatusCode, res.Status))
		}
		buf := &bytes.Buffer{}
		if _, err := buf.ReadFrom(res.Body); err != nil {
			t.Error(err)
		}

		// Compare the body read to the nonce we were expecting.
		t.Logf("\nread:\t%s", buf.String())
		if got, want := buf.String(), tc.Nonce; got != want {
			t.Error(fmt.Errorf("got: %q, want: %q", got, want))
		}
	}
}

func TestMTLS(t *testing.T) {
	t.Parallel()
	ctx := zlog.Test(context.Background(), t)
	t.Run("Basic", new(mtlsTestcase).Run(ctx))
}
