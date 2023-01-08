package caramelmail

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/labstack/echo/v4"
	"github.com/toorop/go-dkim"
	"github.com/xhit/go-simple-mail/v2"
	"log"
	"net"
	"strings"
	"time"
)

type BulkMail struct {
	From       string `json:"from"`
	fromHost   string
	FromName   string `json:"fromName"`
	PrivateKey string `json:"privateKey"`
	ToHost     string
	Mail       []Mail `json:"mail"`
}

func addBulkMail(c echo.Context) error {
	post := new(BulkMail)
	if err := c.Bind(post); err != nil {
		return err
	}

	name, domain, err := splitAddress(post.From)
	if err != nil {
		return err
	}

	if post.FromName == "" {
		post.FromName = name
	}

	post.fromHost = domain

	var mail map[string][]Mail
	for _, m := range post.Mail {
		_, domain, err = splitAddress(m.To)
		if err != nil {
			return err
		}

		if _, ok := mail[domain]; !ok {
			mail[domain] = []Mail{}
		}

		mail[domain] = append(mail[domain], m)
	}

	// slice mails every 100 mails
	for _, mails := range mail {
		mailsLen := len(mails)
		for i := 0; i < mailsLen; i += 100 {
			end := i + 100
			if end > mailsLen {
				end = mailsLen
			}

			set := &BulkMail{
				From:       post.From,
				fromHost:   post.fromHost,
				FromName:   post.FromName,
				PrivateKey: post.PrivateKey,
				ToHost:     post.ToHost,
				Mail:       mails[i:end],
			}

			if message, _ := json.Marshal(set); message != nil {
				if err = singleQueue.Publish(string(message)); err != nil {
					return err
				}
			}
		}
	}

	return nil
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
