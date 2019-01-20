package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/mail"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type MessageResponse struct {
	ID      string
	Subject string
	Date    time.Time
	To      string
}

func (m *MailDam) ListAPI(w http.ResponseWriter, req *http.Request) {
	files, err := ioutil.ReadDir(m.dataDir)
	if err != nil {
		log.Printf("failed to read %s: %s", m.dataDir, err)
		return
	}
	paths := []string{}
	for _, f := range files {
		paths = append(paths, filepath.Join(m.dataDir, f.Name()))
	}

	resp := []MessageResponse{}
	for _, p := range paths {
		b, err := ioutil.ReadFile(p)
		if err != nil {
			log.Printf("failed to read %s: %s", p, err)
			continue
		}
		e := &envelopeRecorder{}
		if err := json.Unmarshal(b, e); err != nil {
			log.Printf("failed to load json from %s: %s", p, err)
			continue
		}
		dec := new(mime.WordDecoder)
		header := e.Header()
		rawSubject := header.Get("Subject")
		subject, err := dec.DecodeHeader(rawSubject)
		if err != nil {
			log.Printf("failed to decode subject: %s: %s", p, err)
			continue
		}
		t, err := mail.ParseDate(header.Get("Date"))
		if err != nil {
			log.Printf("failed to parse date: %s: %s", p, err)
			continue
		}
		resp = append(resp, MessageResponse{
			ID:      e.ID,
			Subject: subject,
			To:      header.Get("To"),
			Date:    t,
		})
	}
	b, _ := json.MarshalIndent(resp, "", "  ")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func (m *MailDam) GetAPI(w http.ResponseWriter, req *http.Request) {
	if strings.HasSuffix(req.URL.Path, "/") {
		return
	}
	id := path.Base(req.URL.Path)
	e, err := OpenEnvelopeRecorder(m.dataDir, id)
	if err != nil {
		log.Printf("failed to open envelope %s: %s", id, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(e.Payload)
}

func (m *MailDam) SendAPI(w http.ResponseWriter, req *http.Request) {
	if strings.HasSuffix(req.URL.Path, "/") {
		return
	}
	id := path.Base(req.URL.Path)
	e, err := OpenEnvelopeRecorder(m.dataDir, id)
	if err != nil {
		log.Printf("failed to open envelope %s: %s", id, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := m.sendmail(e); err != nil {
		log.Printf("failed to send mail %s: %s", id, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := os.Remove(filepath.Join(m.dataDir, id)); err != nil {
		log.Printf("failed to remove %s: %s", id, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	return
}

func (m *MailDam) ListenAndServeAPI() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/list", m.ListAPI)
	mux.HandleFunc("/api/get/", m.GetAPI)
	mux.HandleFunc("/api/send/", m.SendAPI)
	s := &http.Server{
		Addr:           ":8025",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
