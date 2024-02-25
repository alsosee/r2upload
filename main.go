// A go server to upload media to R2 Cloudflare storage
// The only use case is local development,
// since Cloudflare's Wrangler CLI `pages dev` R2 binding does not work with remote storage.
package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/charmbracelet/log"
	flags "github.com/jessevdk/go-flags"
)

type appConfig struct {
	Bind string `env:"BIND" long:"bind" description:"Bind address" default:"localhost:8789"`

	// Cloudflare R2 storage
	R2AccountID       string `env:"R2_ACCOUNT_ID" long:"r2-account-id" description:"r2 account id"`
	R2AccessKeyID     string `env:"R2_ACCESS_KEY_ID" long:"r2-access-key-id" description:"r2 access key id"`
	R2AccessKeySecret string `env:"R2_ACCESS_KEY_SECRET" long:"r2-access-key-secret" description:"r2 access key secret"`
	R2Bucket          string `env:"R2_BUCKET" long:"r2-bucket" description:"r2 bucket"`
}

var cfg appConfig

func main() {
	log.Info("Starting...")

	_, err := flags.Parse(&cfg)
	if err != nil {
		log.Fatalf("Error parsing flags: %v", err)
	}

	r2, err := NewR2(
		cfg.R2AccountID,
		cfg.R2AccessKeyID,
		cfg.R2AccessKeySecret,
		cfg.R2Bucket,
	)
	if err != nil {
		log.Fatalf("Error creating R2: %v", err)
	}

	// start HTTP handler with /upload handler
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			log.Errorf("Method not allowed: %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// read filename from "x-file-name" header
		filename := r.Header.Get("x-file-name")
		if filename == "" {
			log.Errorf("Missing x-file-name header")
			http.Error(w, "Missing x-file-name header", http.StatusBadRequest)
			return
		}

		// read request body
		if r.Body == nil {
			log.Errorf("Missing request body")
			http.Error(w, "Missing request body", http.StatusBadRequest)
			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			log.Errorf("Error reading request body: %v", err)
			http.Error(w, fmt.Sprintf("Error reading request body: %v", err), http.StatusInternalServerError)
			return
		}

		// upload media to R2
		log.Infof("Uploading %s", filename)
		err = r2.Upload(r.Context(), filename, b)
		if err != nil {
			log.Errorf("Error uploading media: %v", err)
			http.Error(w, fmt.Sprintf("Error uploading media: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})

	log.Infof("Listening on %s", cfg.Bind)
	if err = http.ListenAndServe(cfg.Bind, nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
