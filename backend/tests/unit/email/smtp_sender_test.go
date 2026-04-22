package email_test

import (
	"bufio"
	"context"
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
