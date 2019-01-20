package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/mail"
	"path/filepath"

	"github.com/bradfitz/go-smtpd/smtpd"
)

type envelopeRecorder struct {
	HandleReceived func(*envelopeRecorder) `json:"-"`
	ID             string
	From           string
	RCPTS          []string
	Payload        []byte
	buf            bytes.Buffer
}

func (e *envelopeRecorder) AddRecipient(rcpt smtpd.MailAddress) error {
	e.RCPTS = append(e.RCPTS, rcpt.Email())
	return nil
}

func (e *envelopeRecorder) BeginData() error {
	if len(e.RCPTS) == 0 {
		return smtpd.SMTPError("554 5.5.1 Error: no valid recipients")
	}
	return nil
}

func (e *envelopeRecorder) Write(line []byte) error {
	e.buf.Write(line)
	return nil
}

func (e *envelopeRecorder) Close() error {
	e.Payload = e.buf.Bytes()
	e.ID = fmt.Sprintf("%x", sha256.Sum256(e.Payload))
	if e.HandleReceived != nil {
		e.HandleReceived(e)
	}
	return nil
}

func (e *envelopeRecorder) Header() mail.Header {
	r := bytes.NewBuffer(e.Payload)
	m, err := mail.ReadMessage(r)
	if err != nil {
		log.Fatal(err)
	}
	return m.Header
}

func OpenEnvelopeRecorder(dir, id string) (*envelopeRecorder, error) {
	p := filepath.Join(dir, id)
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %s", p, err)
	}
	e := &envelopeRecorder{}
	if err := json.Unmarshal(b, e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json %s: %s", id, err)
	}
	return e, nil
}
