package caramelmail

import (
	"crypto/tls"
	"github.com/toorop/go-dkim"
	mail "github.com/xhit/go-simple-mail/v2"
	"log"
	"net"
	"time"
)

type Mail struct {
	From       string
	FromName   string
	To         string
	ToName     string
	Subject    string
	Body       string
	PrivateKey string
}

func (m *Mail) Send() {
	_, toDomain, err := splitAddress(m.To)
	if err != nil {
		log.Fatal(err)
	}

	_, fromDomain, err := splitAddress(m.From)
	if err != nil {
		log.Fatal(err)
	}

	mxs, err := net.LookupMX(toDomain)
	if err != nil {
		log.Fatal(err)
	}
	if len(mxs) == 0 {
		mxs = []*net.MX{{Host: toDomain}}
	}

	for _, mx := range mxs {
		server := mail.NewSMTPClient()

		// SMTP Server
		server.Host = mx.Host
		server.Port = 25
		server.Encryption = mail.EncryptionSTARTTLS
		server.KeepAlive = true
		server.Authentication = mail.AuthNone
		server.ConnectTimeout = 10 * time.Second
		server.SendTimeout = 10 * time.Second
		server.TLSConfig = &tls.Config{InsecureSkipVerify: true}

		// SMTP client
		smtpClient, err := server.Connect()
		if err != nil {
			log.Fatal(err)
		}

		email := mail.NewMSG()
		email.SetFrom(m.FromName + " <" + m.From + ">").
			AddTo(m.To).
			SetSubject(m.Subject)
		email.SetBody(mail.TextHTML, m.Body)

		if m.PrivateKey != "" {
			options := dkim.NewSigOptions()
			options.PrivateKey = []byte(m.PrivateKey)
			options.Domain = fromDomain
			options.Selector = "default"
			options.SignatureExpireIn = 3600
			options.Headers = []string{"from", "date", "mime-version", "received", "received"}
			options.AddSignatureTimestamp = true
			options.Canonicalization = "relaxed/relaxed"

			email.SetDkim(options)
		}

		if email.Error != nil {
			log.Fatal(email.Error)
		}

		err = email.Send(smtpClient)
		if err != nil {
			log.Println(err)
		} else {
			log.Println("Email Sent")
		}
	}
}
