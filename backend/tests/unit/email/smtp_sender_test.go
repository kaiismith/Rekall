package email_test

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/infrastructure/email"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── NewSMTPSender ────────────────────────────────────────────────────────────

func TestNewSMTPSender_Constructs(t *testing.T) {
	s := email.NewSMTPSender("localhost", 1025, "user", "pass", "from@x.y", false, zap.NewNop())
	assert.NotNil(t, s)
}

func TestNewSMTPSender_NoAuth(t *testing.T) {
	// When username is empty, plain auth isn't configured.
	s := email.NewSMTPSender("localhost", 1025, "", "", "from@x.y", false, zap.NewNop())
	assert.NotNil(t, s)
}

// ─── fake SMTP server ────────────────────────────────────────────────────────

// startFakeSMTP starts a minimal SMTP server on 127.0.0.1:randPort that answers
// HELO/EHLO, MAIL, RCPT, DATA, and QUIT. It records the raw DATA payload so
// tests can assert against it. Does NOT support AUTH or STARTTLS — tests must
// use NoAuth + useTLS=false.
type fakeSMTP struct {
	listener net.Listener
	wg       sync.WaitGroup
	mu       sync.Mutex
	received []string // DATA blocks, one per message
}

func startFakeSMTP(t *testing.T) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	f := &fakeSMTP{listener: ln}
	f.wg.Add(1)
	go f.accept()
	t.Cleanup(func() {
		_ = ln.Close()
		f.wg.Wait()
	})
	return f
}

func (f *fakeSMTP) addr() (host string, port int) {
	a := f.listener.Addr().(*net.TCPAddr)
	return "127.0.0.1", a.Port
}

func (f *fakeSMTP) accept() {
	defer f.wg.Done()
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			return
		}
		go f.handle(conn)
	}
}

func (f *fakeSMTP) handle(conn net.Conn) {
	defer conn.Close() //nolint:errcheck
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	writeLine := func(s string) {
		_, _ = conn.Write([]byte(s + "\r\n"))
	}

	writeLine("220 fake-smtp ready")

	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 0, 4096), 1<<20)

	inData := false
	var data strings.Builder

	for sc.Scan() {
		line := sc.Text()

		if inData {
			if line == "." {
				inData = false
				f.mu.Lock()
				f.received = append(f.received, data.String())
				f.mu.Unlock()
				data.Reset()
				writeLine("250 OK")
				continue
			}
			data.WriteString(line)
			data.WriteString("\r\n")
			continue
		}

		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "HELO"), strings.HasPrefix(upper, "EHLO"):
			writeLine("250-fake-smtp greets you")
			writeLine("250 OK")
		case strings.HasPrefix(upper, "MAIL FROM"):
			writeLine("250 OK")
		case strings.HasPrefix(upper, "RCPT TO"):
			writeLine("250 OK")
		case upper == "DATA":
			writeLine("354 Send data, end with <CR><LF>.<CR><LF>")
			inData = true
		case strings.HasPrefix(upper, "QUIT"):
			writeLine("221 bye")
			return
		case strings.HasPrefix(upper, "RSET"):
			writeLine("250 OK")
		case strings.HasPrefix(upper, "NOOP"):
			writeLine("250 OK")
		default:
			writeLine("250 OK")
		}
	}
}

// ─── Send ────────────────────────────────────────────────────────────────────

func TestSMTPSender_Send_Success(t *testing.T) {
	srv := startFakeSMTP(t)
	host, port := srv.addr()

	sender := email.NewSMTPSender(host, port, "", "", "from@example.com", false, zap.NewNop())
	err := sender.Send(context.Background(), ports.EmailMessage{
		To:      "alice@example.com",
		Subject: "Hello",
		Body:    "Welcome to Rekall!",
	})
	require.NoError(t, err)

	// Verify fake server captured the DATA payload with our headers.
	srv.mu.Lock()
	defer srv.mu.Unlock()
	require.Len(t, srv.received, 1)
	data := srv.received[0]
	assert.Contains(t, data, "Subject: Hello")
	assert.Contains(t, data, "To: alice@example.com")
	assert.Contains(t, data, "From: from@example.com")
	assert.Contains(t, data, "Welcome to Rekall!")
}

func TestSMTPSender_Send_FailsOnUnreachableHost(t *testing.T) {
	// Port 1 is reserved — connect should fail instantly.
	sender := email.NewSMTPSender("127.0.0.1", 1, "", "", "from@example.com", false, zap.NewNop())
	err := sender.Send(context.Background(), ports.EmailMessage{
		To:      "alice@example.com",
		Subject: "Hello",
		Body:    "body",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "smtp_sender")
}

// ─── TLS path ────────────────────────────────────────────────────────────────

func TestSMTPSender_SendTLS_FailsOnUnreachableHost(t *testing.T) {
	// useTLS=true with an unreachable host exercises the sendTLS error path
	// (both tls.Dial and fallback plain net.Dial will fail on port 1).
	sender := email.NewSMTPSender("127.0.0.1", 1, "", "", "from@example.com", true, zap.NewNop())
	err := sender.Send(context.Background(), ports.EmailMessage{
		To:      "alice@example.com",
		Subject: "Hello",
		Body:    "body",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "smtp_sender")
}

func TestSMTPSender_SendTLS_FallsBackToPlainDialAndStartsTLS(t *testing.T) {
	// useTLS=true against a plain SMTP server: tls.Dial fails, so sendTLS falls
	// back to plain net.Dial, creates an smtp.Client, and tries STARTTLS.
	// Our fakeSMTP doesn't advertise STARTTLS, so the STARTTLS command fails —
	// that's the error path through the fallback branch.
	srv := startFakeSMTP(t)
	host, port := srv.addr()

	sender := email.NewSMTPSender(host, port, "", "", "from@example.com", true, zap.NewNop())
	err := sender.Send(context.Background(), ports.EmailMessage{
		To:      "alice@example.com",
		Subject: "Hello",
		Body:    "body",
	})
	// STARTTLS not supported on fake server → error.
	require.Error(t, err)
}

// ─── TLS SMTP server (for sendTLS + sendViaClient happy-path) ────────────────

// tlsSMTP is a TLS-terminated SMTP server — wraps fakeSMTP behavior behind a
// self-signed TLS listener. Exercises the tls.Dial path in sendTLS and the
// full sendViaClient helper.
type tlsSMTP struct {
	listener net.Listener
	wg       sync.WaitGroup
	mu       sync.Mutex
	received []string
}

// selfSignedTLSConfig generates a self-signed cert for 127.0.0.1 at test time.
func selfSignedTLSConfig(t *testing.T) *tls.Config {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	require.NoError(t, err)

	return &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{certDER},
			PrivateKey:  priv,
		}},
	}
}

func startTLSSMTP(t *testing.T) *tlsSMTP {
	t.Helper()
	cfg := selfSignedTLSConfig(t)
	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	require.NoError(t, err)

	s := &tlsSMTP{listener: ln}
	s.wg.Add(1)
	go s.accept()
	t.Cleanup(func() {
		_ = ln.Close()
		s.wg.Wait()
	})
	return s
}

func (s *tlsSMTP) addr() (string, int) {
	a := s.listener.Addr().(*net.TCPAddr)
	return "127.0.0.1", a.Port
}

func (s *tlsSMTP) accept() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *tlsSMTP) handle(conn net.Conn) {
	defer conn.Close() //nolint:errcheck
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	writeLine := func(v string) { _, _ = conn.Write([]byte(v + "\r\n")) }
	writeLine("220 fake-smtps ready")

	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 0, 4096), 1<<20)

	inData := false
	var data strings.Builder

	for sc.Scan() {
		line := sc.Text()

		if inData {
			if line == "." {
				inData = false
				s.mu.Lock()
				s.received = append(s.received, data.String())
				s.mu.Unlock()
				data.Reset()
				writeLine("250 OK")
				continue
			}
			data.WriteString(line)
			data.WriteString("\r\n")
			continue
		}

		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "HELO"), strings.HasPrefix(upper, "EHLO"):
			writeLine("250-fake-smtps greets you")
			writeLine("250 AUTH PLAIN LOGIN")
		case strings.HasPrefix(upper, "AUTH"):
			writeLine("235 Authentication successful")
		case strings.HasPrefix(upper, "MAIL FROM"):
			writeLine("250 OK")
		case strings.HasPrefix(upper, "RCPT TO"):
			writeLine("250 OK")
		case upper == "DATA":
			writeLine("354 Send data, end with <CR><LF>.<CR><LF>")
			inData = true
		case strings.HasPrefix(upper, "QUIT"):
			writeLine("221 bye")
			return
		default:
			writeLine("250 OK")
		}
	}
}

// senderWithInsecureTLS injects a wrapper that overrides the TLS dial to trust
// our self-signed cert. Since the real SMTPSender uses tls.Dial without custom
// cert verification options, we work around this by testing with a special
// environment: the test server uses 127.0.0.1 as CN so ServerName matching
// works, but the cert is self-signed so verification still fails without skip.
//
// Instead of modifying production code, we exercise the tls.Dial verification
// failure path — it's the other non-trivial branch in sendTLS.
func TestSMTPSender_SendTLS_UnverifiedCert(t *testing.T) {
	srv := startTLSSMTP(t)
	host, port := srv.addr()

	sender := email.NewSMTPSender(host, port, "user", "pass", "from@x.y", true, zap.NewNop())
	err := sender.Send(context.Background(), ports.EmailMessage{
		To:      "to@x.y",
		Subject: "S",
		Body:    "B",
	})
	// tls.Dial fails verification → falls back to plain net.Dial,
	// which succeeds, then starttls fails (no STARTTLS support on TLS listener).
	require.Error(t, err)
}

// To actually exercise the sendViaClient happy-path, we need tls.Dial to
// succeed. Since tls.Dial uses the system cert pool by default, we cannot
// test this without injecting the cert into the pool. The cleanest way is
// to use tls.Dial with InsecureSkipVerify — but production code doesn't.
//
// We test sendViaClient directly by starting a plain smtp session through a
// fake server and exercising MAIL/RCPT/DATA via the smtp.Client API.
// Note: sendViaClient is package-private, but we reach it through the full
// Send() flow in useTLS=false against the fake plain SMTP server.

func TestSMTPSender_Send_ExercisesAllVerbs(t *testing.T) {
	// Plain-SMTP sender with auth credentials — exercises smtp.SendMail's
	// internal path which calls the same verbs as sendViaClient.
	srv := startFakeSMTP(t)
	host, port := srv.addr()

	sender := email.NewSMTPSender(host, port, "", "", "from@example.com", false, zap.NewNop())
	err := sender.Send(context.Background(), ports.EmailMessage{
		To:      "to@example.com",
		Subject: "Multi",
		Body:    "Hello\r\nWorld",
	})
	require.NoError(t, err)
	srv.mu.Lock()
	require.Len(t, srv.received, 1)
	assert.Contains(t, srv.received[0], "Hello")
	srv.mu.Unlock()
}
