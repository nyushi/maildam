package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/smtp"
	"os"
	"strings"

	"github.com/bradfitz/go-smtpd/smtpd"
	"github.com/k0kubun/pp"
)

// OnNewMail is callback for smtpd.Server
func (m *MailDam) OnNewMail(c smtpd.Connection, from smtpd.MailAddress) (smtpd.Envelope, error) {
	pp.Println(from)
	if from.Email() != m.MyEmail {
		return nil, fmt.Errorf("%s is not my account", from.Email())
	}

	e := &envelopeRecorder{
		From:           from.Email(),
		HandleReceived: m.OnMailReceived,
	}
	return e, nil
}

// OnMailReceived is callback function for envelopeRecorder
func (m *MailDam) OnMailReceived(e *envelopeRecorder) {
	pp.Println(e.RCPTS)
	fmt.Println(string(e.Payload))
	b, err := json.Marshal(e)
	if err != nil {
		log.Printf("failed to marshal json: %s", err)
		return
	}
	if err := ioutil.WriteFile(fmt.Sprintf("%s/%s", m.dataDir, e.ID), b, 0644); err != nil {
		log.Printf("failed to write file: %s", err)
	}
}

func (m *MailDam) sendmail(e *envelopeRecorder) error {
	auth := smtp.PlainAuth("", m.SMTPAccount, m.SMTPPassword, m.SMTPServer)
	if err := smtp.SendMail(fmt.Sprintf("%s:%s", m.SMTPServer, m.SMTPPort), auth, e.From, e.RCPTS, e.Payload); err != nil {
		return fmt.Errorf("failed to send mail: %s", err)
	}
	return nil
}

// MailDam represents App configuration
type MailDam struct {
	MyEmail      string
	SMTPServer   string
	SMTPPort     string
	SMTPAccount  string
	SMTPPassword string

	dataDir string
}

func main() {
	dataDir := strings.TrimSuffix("./data", "/")
	os.MkdirAll(dataDir, os.ModePerm)
	m := MailDam{
		MyEmail:      os.Getenv("MD_MY_EMAIL"),
		SMTPServer:   os.Getenv("MD_SMTP_SERVER"),
		SMTPPort:     os.Getenv("MD_SMTP_PORT"),
		SMTPAccount:  os.Getenv("MD_SMTP_USER"),
		SMTPPassword: os.Getenv("MD_SMTP_PASS"),
		dataDir:      os.Getenv("MD_DATA_DIR"),
	}
	pp.Println(m)
	s := smtpd.Server{
		OnNewMail: m.OnNewMail,
	}
	go m.ListenAndServeAPI()
	log.Println(s.ListenAndServe())
}
