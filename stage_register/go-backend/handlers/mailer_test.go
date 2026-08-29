package handlers

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
)

func TestIsProductionRecognizesStatGateEnv(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("GO_ENV", "")
	t.Setenv("STATGATE_ENV", "production")
	if !isProduction() {
		t.Fatal("expected STATGATE_ENV=production to enable production mode")
	}
}

func TestSendRegistryEmailRequiresSMTPConfiguration(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	if err := sendRegistryEmail("user@example.test", "Subject", "Body"); err == nil {
		t.Fatal("expected missing SMTP_HOST to fail")
	}
}

func TestSendRegistryEmailDeliversThroughSMTP(t *testing.T) {
	addr, messages, stop := startFakeSMTP(t)
	defer stop()

	host, port, _ := strings.Cut(addr, ":")
	t.Setenv("SMTP_HOST", host)
	t.Setenv("SMTP_PORT", port)
	t.Setenv("SMTP_FROM", "noreply@statgate.test")
	t.Setenv("SMTP_USER", "")
	t.Setenv("SMTP_PASSWORD", "")

	if err := sendRegistryEmail("recipient@statgate.test", "Verify your StatGate email", "Hello from StatGate"); err != nil {
		t.Fatalf("send email: %v", err)
	}

	message := <-messages
	if !strings.Contains(message, "To: recipient@statgate.test") {
		t.Fatalf("message missing recipient header: %q", message)
	}
	if !strings.Contains(message, "Subject: Verify your StatGate email") {
		t.Fatalf("message missing subject: %q", message)
	}
	if !strings.Contains(message, "Hello from StatGate") {
		t.Fatalf("message missing body: %q", message)
	}
}

func startFakeSMTP(t *testing.T) (string, <-chan string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	messages := make(chan string, 1)
	done := make(chan struct{})

	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		writeLine := func(line string) {
			_, _ = fmt.Fprintf(conn, "%s\r\n", line)
		}
		writeLine("220 statgate-test ESMTP")
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			command := strings.TrimSpace(line)
			upper := strings.ToUpper(command)
			switch {
			case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
				writeLine("250-statgate-test")
				writeLine("250 OK")
			case strings.HasPrefix(upper, "MAIL FROM:"):
				writeLine("250 OK")
			case strings.HasPrefix(upper, "RCPT TO:"):
				writeLine("250 OK")
			case upper == "DATA":
				writeLine("354 End data with <CR><LF>.<CR><LF>")
				var builder strings.Builder
				for {
					dataLine, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					if strings.TrimSpace(dataLine) == "." {
						break
					}
					builder.WriteString(dataLine)
				}
				messages <- builder.String()
				writeLine("250 OK")
			case upper == "QUIT":
				writeLine("221 Bye")
				return
			default:
				writeLine("250 OK")
			}
		}
	}()

	stop := func() {
		_ = listener.Close()
		<-done
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("SMTP_HOST")
		_ = os.Unsetenv("SMTP_PORT")
		_ = os.Unsetenv("SMTP_FROM")
	})
	return listener.Addr().String(), messages, stop
}
