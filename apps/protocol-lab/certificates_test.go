package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

func testCertificate(t *testing.T, name string, until time.Time) (string, string, []byte, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	uri, _ := url.Parse("urn:otio:test:" + name)
	template := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: name}, NotBefore: time.Now().Add(-48 * time.Hour), NotAfter: until, URIs: []*url.URL{uri}, DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})), der, key
}
func TestCertificateUploadValidationAndMetadata(t *testing.T) {
	t.Setenv("OTIO_CERTIFICATE_DIR", t.TempDir())
	t.Setenv("OTIO_CONFIG_TOKEN", "token")
	cert, key, _, _ := testCertificate(t, "client", time.Now().Add(10*24*time.Hour))
	peer, _, _, _ := testCertificate(t, "server", time.Now().Add(20*24*time.Hour))
	p := certificateProfile{"plant", cert, key, peer}
	put := func(profile certificateProfile, token, origin string) *httptest.ResponseRecorder {
		data, _ := json.Marshal(profile)
		r := httptest.NewRequest("PUT", "http://localhost/api/certificates", bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-OTIO-Config-Token", token)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		certificatesAPI(w, r)
		return w
	}
	if w := put(p, "", "http://localhost"); w.Code != 401 {
		t.Fatalf("missing token: %d", w.Code)
	}
	if w := put(p, "token", "http://other"); w.Code != 403 {
		t.Fatalf("cross-origin: %d", w.Code)
	}
	if w := put(p, "token", "http://localhost"); w.Code != 200 {
		t.Fatalf("upload: %d %s", w.Code, w.Body.String())
	}
	saved, err := loadCertificateProfile("plant")
	if err != nil || saved.PrivateKey != key {
		t.Fatal("persistent profile missing", err)
	}
	w := httptest.NewRecorder()
	certificatesAPI(w, httptest.NewRequest("GET", "/api/certificates", nil))
	if strings.Contains(w.Body.String(), "PRIVATE KEY") || strings.Contains(w.Body.String(), key) {
		t.Fatal("private key leaked")
	}
	bad := p
	bad.ID = "../escape"
	if put(bad, "token", "").Code != 422 {
		t.Fatal("accepted traversal")
	}
	bad = p
	bad.PrivateKey = "invalid"
	if put(bad, "token", "").Code != 422 {
		t.Fatal("accepted invalid key")
	}
	expired, expiredKey, _, _ := testCertificate(t, "expired", time.Now().Add(-time.Hour))
	p.Certificate = expired
	p.PrivateKey = expiredKey
	if put(p, "token", "").Code != 200 {
		t.Fatal("replacement failed")
	}
	metadata, err := profileMetadata(p)
	if err != nil || metadata["valid"] != false {
		t.Fatal("expiry not reported", err)
	}
	if _, _, err := opcuaClientOptions(context.Background(), "opc.tcp://localhost:4840?certificateProfile=plant"); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatal("expired profile did not fail closed", err)
	}
}
func TestOPCUACertificateReadAndPinning(t *testing.T) {
	t.Setenv("OTIO_CERTIFICATE_DIR", t.TempDir())
	cert, key, _, _ := testCertificate(t, "client", time.Now().Add(24*time.Hour))
	peer, _, der, serverKey := testCertificate(t, "server", time.Now().Add(24*time.Hour))
	p := certificateProfile{"secure", cert, key, peer}
	data, _ := json.Marshal(p)
	if err := os.WriteFile(filepath.Join(certificateDir(), "secure.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	srv := server.New(server.EndPoint("127.0.0.1", port), server.Certificate(der), server.PrivateKey(serverKey), server.EnableSecurity("Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt), server.EnableAuthMode(ua.UserTokenTypeAnonymous))
	ns := server.NewNodeNameSpace(srv, "secure-test")
	root, _ := srv.Namespace(0)
	root.Objects().AddRef(ns.Objects(), id.HasComponent, true)
	node := ns.AddNewVariableStringNode("Temperature", func() *ua.DataValue { return server.DataValueFromValue(24.5) })
	ns.Objects().AddRef(node, id.HasComponent, true)
	if err := srv.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	connection := fmt.Sprintf("opc.tcp://127.0.0.1:%d?certificateProfile=secure", port)
	value, err := readValue(ctx, ReadRequest{"opcua-tcp", connection, "ns=1;s=Temperature"})
	if err != nil {
		t.Fatal(err)
	}
	if value != 24.5 {
		t.Fatalf("value=%v", value)
	}
	wrong, _, _, _ := testCertificate(t, "other-server", time.Now().Add(24*time.Hour))
	p.ServerCertificate = wrong
	data, _ = json.Marshal(p)
	os.WriteFile(filepath.Join(certificateDir(), "secure.json"), data, 0600)
	if _, err = readValue(ctx, ReadRequest{"opcua-tcp", connection, "ns=1;s=Temperature"}); err == nil || !strings.Contains(err.Error(), "no insecure fallback") {
		t.Fatal("untrusted peer accepted", err)
	}
}

func TestCertificateMonitoringRejectsCorruptProfiles(t *testing.T) {
	t.Setenv("OTIO_CERTIFICATE_DIR", t.TempDir())
	if err := os.WriteFile(filepath.Join(certificateDir(), "broken.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	certificatesAPI(w, httptest.NewRequest("GET", "/api/certificates", nil))
	if w.Code != 500 {
		t.Fatalf("monitoring silently omitted invalid profile: %d", w.Code)
	}
}
func TestCertificateUploadRejectsTrailingJSON(t *testing.T) {
	t.Setenv("OTIO_CERTIFICATE_DIR", t.TempDir())
	t.Setenv("OTIO_CONFIG_TOKEN", "")
	r := httptest.NewRequest("PUT", "/api/certificates", strings.NewReader(`{} {}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	certificatesAPI(w, r)
	if w.Code != 400 {
		t.Fatalf("trailing input accepted: %d", w.Code)
	}
}
