// Package mailer mengirim email plaintext lewat SMTP untuk undangan event
// terjadwal. Sengaja minimalis (net/smtp + STARTTLS): tidak ada template HTML,
// tidak ada antrian — penjadwalan & retry diurus service pemanggil.
package mailer

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Message adalah satu email yang akan dikirim.
type Message struct {
	FromName  string // nama tampilan pengirim (mis. nama user yang mengundang)
	FromEmail string // alamat yang tampil di header From/Reply-To
	To        []string
	Subject   string
	Body      string // plaintext (undangan formal)
}

// Mailer adalah kemampuan mengirim email — antarmuka agar service bisa di-stub
// di test dan agar "SMTP tidak dikonfigurasi" jadi tipe nil yang jelas.
type Mailer interface {
	Send(msg Message) error
}

// SMTPMailer mengirim lewat SMTP relay yang dikonfigurasi operator via env.
type SMTPMailer struct {
	Host     string
	Port     int
	Username string
	Password string
	// EnvelopeFrom adalah alamat akun relay (MAIL FROM). Header From memakai
	// FromEmail pesan agar undangan tampak datang dari yang mengundang.
	EnvelopeFrom string
}

// addr mengembalikan host:port.
func (m *SMTPMailer) addr() string { return net.JoinHostPort(m.Host, fmt.Sprint(m.Port)) }

// Send menyusun pesan RFC 5322 sederhana dan mengirimnya. Port 465 memakai TLS
// implisit; selain itu memakai STARTTLS bila server mendukung.
func (m *SMTPMailer) Send(msg Message) error {
	if len(msg.To) == 0 {
		return fmt.Errorf("mailer: tidak ada penerima")
	}
	raw := m.build(msg)

	if m.Port == 465 {
		return m.sendImplicitTLS(msg.To, raw)
	}
	return m.sendSTARTTLS(msg.To, raw)
}

// build merangkai header + body menjadi satu pesan mentah (CRLF).
func (m *SMTPMailer) build(msg Message) []byte {
	from := msg.FromEmail
	if from == "" {
		from = m.EnvelopeFrom
	}
	var b strings.Builder
	if msg.FromName != "" {
		fmt.Fprintf(&b, "From: %s <%s>\r\n", encodeHeader(msg.FromName), from)
	} else {
		fmt.Fprintf(&b, "From: %s\r\n", from)
	}
	fmt.Fprintf(&b, "To: %s\r\n", strings.Join(msg.To, ", "))
	if from != "" {
		fmt.Fprintf(&b, "Reply-To: %s\r\n", from)
	}
	fmt.Fprintf(&b, "Subject: %s\r\n", encodeHeader(msg.Subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(msg.Body, "\n", "\r\n"))
	return []byte(b.String())
}

// auth mengembalikan PlainAuth bila kredensial diisi, else nil (relay terbuka).
func (m *SMTPMailer) auth() smtp.Auth {
	if m.Username == "" {
		return nil
	}
	return smtp.PlainAuth("", m.Username, m.Password, m.Host)
}

func (m *SMTPMailer) sendSTARTTLS(to []string, raw []byte) error {
	c, err := smtp.Dial(m.addr())
	if err != nil {
		return fmt.Errorf("mailer: dial: %w", err)
	}
	defer func() { _ = c.Close() }()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: m.Host}); err != nil {
			return fmt.Errorf("mailer: starttls: %w", err)
		}
	}
	if a := m.auth(); a != nil {
		if err := c.Auth(a); err != nil {
			return fmt.Errorf("mailer: auth: %w", err)
		}
	}
	return m.deliver(c, to, raw)
}

func (m *SMTPMailer) sendImplicitTLS(to []string, raw []byte) error {
	conn, err := tls.Dial("tcp", m.addr(), &tls.Config{ServerName: m.Host})
	if err != nil {
		return fmt.Errorf("mailer: tls dial: %w", err)
	}
	c, err := smtp.NewClient(conn, m.Host)
	if err != nil {
		return fmt.Errorf("mailer: client: %w", err)
	}
	defer func() { _ = c.Close() }()
	if a := m.auth(); a != nil {
		if err := c.Auth(a); err != nil {
			return fmt.Errorf("mailer: auth: %w", err)
		}
	}
	return m.deliver(c, to, raw)
}

// deliver menjalankan MAIL/RCPT/DATA untuk satu pesan ke banyak penerima.
func (m *SMTPMailer) deliver(c *smtp.Client, to []string, raw []byte) error {
	if err := c.Mail(m.EnvelopeFrom); err != nil {
		return fmt.Errorf("mailer: MAIL FROM: %w", err)
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("mailer: RCPT %s: %w", rcpt, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("mailer: DATA: %w", err)
	}
	if _, err := w.Write(raw); err != nil {
		return fmt.Errorf("mailer: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mailer: close body: %w", err)
	}
	return c.Quit()
}

// encodeHeader membungkus nilai header non-ASCII sebagai MIME encoded-word agar
// nama/subjek berhuruf non-Latin tetap benar. ASCII murni dilewatkan apa adanya.
func encodeHeader(s string) string {
	for _, r := range s {
		if r > 127 {
			return "=?UTF-8?B?" + base64Encode(s) + "?="
		}
	}
	return s
}

func base64Encode(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
