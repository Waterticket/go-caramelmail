package caramelmail

import (
	"crypto/tls"
	"errors"
	"github.com/toorop/go-dkim"
	"github.com/xhit/go-simple-mail/v2"
	"log"
	"net"
	"strings"
	"time"
)

type BulkMail struct {
	From       string
	fromHost   string
	FromName   string
	PrivateKey string
	ToHost     string
	Mail       []Mail
}

func NewBulkMail(from, fromName, privateKey, toHost string, mail []Mail) (*BulkMail, error) {
	_, domain, err := splitAddress(from)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &BulkMail{
		From:       from,
		fromHost:   domain,
		FromName:   fromName,
		PrivateKey: privateKey,
		ToHost:     toHost,
		Mail:       mail,
	}, nil
}

func (b *BulkMail) Send() {
	mxs, err := net.LookupMX(b.ToHost)
	if err != nil {
		log.Fatal(err)
	}
	if len(mxs) == 0 {
		mxs = []*net.MX{{Host: b.ToHost}}
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

		var options dkim.SigOptions

		if b.PrivateKey != "" {
			options = dkim.NewSigOptions()
			options.PrivateKey = []byte(b.PrivateKey)
			options.Domain = b.fromHost
			options.Selector = "default"
			options.SignatureExpireIn = 3600
			options.Headers = []string{"from", "date", "mime-version", "received", "received"}
			options.AddSignatureTimestamp = true
			options.Canonicalization = "relaxed/relaxed"
		}

		for _, m := range b.Mail {
			email := mail.NewMSG()
			email.SetFrom(b.FromName + " <" + b.From + ">").
				AddTo(m.To).
				SetSubject(m.Subject)
			email.SetBody(mail.TextHTML, m.Body)

			if b.PrivateKey != "" {
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
}

func splitAddress(addr string) (local, domain string, err error) {
	parts := strings.SplitN(addr, "@", 2)
	if len(parts) != 2 {
		return "", "", errors.New("mta: invalid mail address")
	}
	return parts[0], parts[1], nil
}
