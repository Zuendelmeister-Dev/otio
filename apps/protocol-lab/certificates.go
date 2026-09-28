package main

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// Profiles are loaded per connection so certificate replacement takes effect on the next read.
// Private keys never appear in listing responses or generated configuration snippets.
type certificateProfile struct {
	ID                string `json:"id"`
	Certificate       string `json:"certificate"`
	PrivateKey        string `json:"privateKey"`
	ServerCertificate string `json:"serverCertificate"`
}

var profileID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
var certificateMu sync.Mutex

func certificateDir() string { return os.Getenv("OTIO_CERTIFICATE_DIR") }
func parseCertificate(value string) (*x509.Certificate, error) {
	block, rest := pem.Decode([]byte(value))
	if block == nil || block.Type != "CERTIFICATE" || len(bytes.TrimSpace(rest)) != 0 {
		return nil, errors.New("upload exactly one PEM certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}
func validateProfile(p certificateProfile) (*x509.Certificate, *x509.Certificate, *rsa.PrivateKey, error) {
	if !profileID.MatchString(p.ID) {
		return nil, nil, nil, errors.New("profile ID: use 1–64 letters, numbers, hyphens or underscores")
	}
	client, err := parseCertificate(p.Certificate)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("client certificate: %w", err)
	}
	server, err := parseCertificate(p.ServerCertificate)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("server certificate: %w", err)
	}
	pair, err := tls.X509KeyPair([]byte(p.Certificate), []byte(p.PrivateKey))
	if err != nil {
		return nil, nil, nil, errors.New("client certificate and unencrypted PEM private key do not match")
	}
	key, ok := pair.PrivateKey.(*rsa.PrivateKey)
	if !ok || key.N.BitLen() < 2048 {
		return nil, nil, nil, errors.New("OPC UA Basic256Sha256 requires an RSA key of at least 2048 bits")
	}
	if len(client.URIs) == 0 {
		return nil, nil, nil, errors.New("client certificate requires an OPC UA Application URI in its subject alternative names")
	}
	if _, ok := server.PublicKey.(*rsa.PublicKey); !ok {
		return nil, nil, nil, errors.New("server certificate must use RSA")
	}
	return client, server, key, nil
}
func validNow(c *x509.Certificate) bool {
	now := time.Now()
	return !now.Before(c.NotBefore) && now.Before(c.NotAfter)
}
func profileMetadata(p certificateProfile) (map[string]any, error) {
	client, server, _, err := validateProfile(p)
	if err != nil {
		return nil, err
	}
	expiry := client.NotAfter
	if server.NotAfter.Before(expiry) {
		expiry = server.NotAfter
	}
	fingerprint := sha256.Sum256(server.Raw)
	return map[string]any{"id": p.ID, "purpose": "OPC UA · Basic256Sha256 / SignAndEncrypt", "clientSubject": client.Subject.String(), "clientExpires": client.NotAfter, "serverExpires": server.NotAfter, "expires": expiry, "daysRemaining": int(time.Until(expiry).Hours() / 24), "valid": validNow(client) && validNow(server), "serverFingerprint": hex.EncodeToString(fingerprint[:])}, nil
}
func loadCertificateProfile(id string) (certificateProfile, error) {
	var p certificateProfile
	if !profileID.MatchString(id) || certificateDir() == "" {
		return p, errors.New("certificate profile is unavailable; configure OTIO_CERTIFICATE_DIR")
	}
	data, err := os.ReadFile(filepath.Join(certificateDir(), id+".json"))
	if err != nil {
		return p, errors.New("certificate profile not found on this gateway")
	}
	if err = json.Unmarshal(data, &p); err != nil {
		return p, errors.New("invalid stored certificate profile")
	}
	if p.ID != id {
		return p, errors.New("certificate profile identity mismatch")
	}
	return p, nil
}
func certificatesAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodGet {
		items := []map[string]any{}
		if certificateDir() == "" {
			writeJSON(w, 200, map[string]any{"items": items, "enabled": false})
			return
		}
		files, err := os.ReadDir(certificateDir())
		if err != nil && !os.IsNotExist(err) {
			writeJSON(w, 500, map[string]any{"error": "certificate directory unavailable"})
			return
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			p, e := loadCertificateProfile(strings.TrimSuffix(f.Name(), ".json"))
			if e != nil {
				writeJSON(w, 500, map[string]any{"error": "stored certificate profile unreadable; monitoring incomplete"})
				return
			}
			item, e := profileMetadata(p)
			if e != nil {
				writeJSON(w, 500, map[string]any{"error": "stored certificate profile invalid; monitoring incomplete"})
				return
			}
			items = append(items, item)
		}
		writeJSON(w, 200, map[string]any{"items": items, "enabled": true})
		return
	}
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if token := os.Getenv("OTIO_CONFIG_TOKEN"); token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(r.Header.Get("X-OTIO-Config-Token"))) != 1 {
		writeJSON(w, 401, map[string]any{"error": "configuration token required"})
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host != r.Host {
			writeJSON(w, 403, map[string]any{"error": "cross-origin certificate upload rejected"})
			return
		}
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJSON(w, 415, map[string]any{"error": "JSON required"})
		return
	}
	if certificateDir() == "" {
		writeJSON(w, 503, map[string]any{"error": "configure a persistent OTIO_CERTIFICATE_DIR on Protocol Lab first"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 256*1024)
	var p certificateProfile
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		writeJSON(w, 400, map[string]any{"error": "invalid or oversized certificate upload"})
		return
	}
	if decoder.Decode(new(any)) != io.EOF {
		writeJSON(w, 400, map[string]any{"error": "upload must contain one JSON object"})
		return
	}
	metadata, err := profileMetadata(p)
	if err != nil {
		writeJSON(w, 422, map[string]any{"error": err.Error()})
		return
	}
	certificateMu.Lock()
	defer certificateMu.Unlock()
	if err = os.MkdirAll(certificateDir(), 0700); err != nil {
		writeJSON(w, 500, map[string]any{"error": "cannot create certificate storage"})
		return
	}
	data, _ := json.Marshal(p)
	tmp, err := os.CreateTemp(certificateDir(), ".upload-")
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": "cannot write certificate storage"})
		return
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(data)
	}
	closeErr := tmp.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(name, filepath.Join(certificateDir(), p.ID+".json"))
	}
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": "could not persist certificate profile"})
		return
	}
	writeJSON(w, 200, metadata)
}
func opcuaClientOptions(ctx context.Context, connection string) (string, []opcua.Option, error) {
	u, err := url.Parse(connection)
	if err != nil {
		return "", nil, err
	}
	q := u.Query()
	id := q.Get("certificateProfile")
	q.Del("certificateProfile")
	u.RawQuery = q.Encode()
	endpoint := u.String()
	if id == "" {
		return endpoint, []opcua.Option{opcua.SecurityMode(ua.MessageSecurityModeNone), opcua.SecurityPolicy(ua.SecurityPolicyURINone), opcua.AutoReconnect(false)}, nil
	}
	p, err := loadCertificateProfile(id)
	if err != nil {
		return "", nil, err
	}
	client, server, key, err := validateProfile(p)
	if err != nil {
		return "", nil, err
	}
	if !validNow(client) || !validNow(server) {
		return "", nil, errors.New("certificate expired or not yet valid; replace the profile in Lense Certificates")
	}
	endpoints, err := opcua.GetEndpoints(ctx, endpoint)
	if err != nil {
		return "", nil, err
	}
	var selected *ua.EndpointDescription
	for _, candidate := range endpoints {
		if candidate.SecurityMode == ua.MessageSecurityModeSignAndEncrypt && candidate.SecurityPolicyURI == ua.SecurityPolicyURIBasic256Sha256 && bytes.Equal(candidate.ServerCertificate, server.Raw) {
			selected = candidate
			break
		}
	}
	if selected == nil {
		return "", nil, errors.New("server certificate or secure endpoint does not match the trusted profile; no insecure fallback")
	}
	return endpoint, []opcua.Option{opcua.SecurityFromEndpoint(selected, ua.UserTokenTypeAnonymous), opcua.Certificate(client.Raw), opcua.PrivateKey(key), opcua.AutoReconnect(false)}, nil
}
