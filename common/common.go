package common

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"time"
)

const (
	TimeLayout1             = "2006-01-02 15:04"
	TimeLayout2             = "15:04:05"
	TimeLayoutMysqlDateTime = "2006-01-02 15:04:05"
)

func Md5Check(hashedStr string, plain string) bool {
	return Md5Hash(plain) == hashedStr
}
func Md5Hash(plain string) string {
	sb := []byte(plain)
	b := md5.Sum(sb)
	return hex.EncodeToString(b[:])
}

type Error struct {
	Err string
}

func (e Error) Error() string {
	return e.Err
}

func ConvUtcToLocal(utcTime string, rawLayout string, newLayout string) string {
	if t, err := time.Parse(rawLayout, utcTime); err != nil {
		panic(err)
	} else {
		return t.Local().Format(newLayout)
	}
}

type EmailSender struct {
	UserName string
	Password string
	Addr     string
	Name     string
}

func (sender EmailSender) SendEmail(toAddress string, subject string, body string) {
	from := mail.Address{sender.Name, sender.UserName}
	to := mail.Address{"", toAddress}

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = from.String()
	headers["To"] = to.String()
	headers["Subject"] = subject
	headers["MIME-version"] = ": 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// Connect to the SMTP Server
	servername := sender.Addr

	host, _, _ := net.SplitHostPort(servername)

	auth := smtp.PlainAuth("", sender.UserName, sender.Password, host)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}

	conn, err := tls.Dial("tcp", servername, tlsconfig)
	if err != nil {
		panic(err)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		panic(err)
	}

	// Auth
	if err = c.Auth(auth); err != nil {
		panic(err)
	}

	// To && From
	if err = c.Mail(from.Address); err != nil {
		panic(err)
	}

	if err = c.Rcpt(to.Address); err != nil {
		panic(err)
	}

	// Data
	w, err := c.Data()
	if err != nil {
		panic(err)
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		panic(err)
	}

	err = w.Close()
	if err != nil {
		panic(err)
	}

	c.Quit()
}
