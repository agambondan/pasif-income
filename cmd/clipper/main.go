package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/agambondan/pasif-income/internal/adapters"
	"github.com/agambondan/pasif-income/internal/core/ports"
	"github.com/agambondan/pasif-income/internal/services"
)

func main() {
	log.Println("--- Podcast Clips Factory Starting ---")

	// 1. Initialize Adapters
	apiKey := os.Getenv("GEMINI_API_KEY")

	downloader := adapters.NewYtdlpDownloader()

	var transcriber ports.Transcriber
	var agent ports.StrategistAgent
	if os.Getenv("USE_MOCK") == "true" {
		transcriber = adapters.NewMockTranscriber()
		agent = adapters.NewMockStrategist()
	} else {
		transcriber = adapters.NewWhisperTranscriber(whisperURL())
		agent = adapters.NewGeminiAgent(apiKey)
	}

	vision := adapters.NewPythonVisionAgent("scripts/face_tracker.py")

	editor := adapters.NewFFmpegEditor()

	// Storage (MinIO)
	storage, err := adapters.NewMinIOStorage(minioEndpoint(), minioAccessKey(), minioSecretKey(), minioBucket())
	if err != nil {
		log.Printf("MinIO Warning: %v (Continuing...)\n", err)
	}

	// Repository (Postgres)
	repo, err := adapters.NewPostgresRepository(postgresDSN())
	if err != nil {
		log.Printf("Postgres Warning: %v (Continuing...)\n", err)
	}

	// 2. Initialize Service
	workflow := services.NewWorkflowService(downloader, transcriber, agent, editor, vision, storage, repo)

	// 3. Optional Background Pipeline Run
	videoURL := os.Getenv("VIDEO_URL")
	if videoURL != "" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
			defer cancel()
			log.Printf("Starting background pipeline for: %s\n", videoURL)
			if err := workflow.RunPipeline(ctx, videoURL); err != nil {
				log.Printf("Pipeline error: %v\n", err)
			}
		}()
	}

	// 4. HTTP API for Dashboard
	http.HandleFunc("/api/clips", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "GET" {
			clips, err := repo.ListClips(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(clips)
			return
		}

		if r.Method == "PATCH" {
			var update struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			}
			if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := repo.UpdateStatus(r.Context(), update.ID, update.Status); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	port := os.Getenv("CLIPPER_PORT")
	if port == "" {
		port = ":8081"
	}
	if port[0] != ':' {
		port = ":" + port
	}

	log.Printf("Listening on %s for clipper dashboard...", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func postgresDSN() string {
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return dsn
	}
	return "postgres://factory:secretpassword@localhost:5432/clips_db?sslmode=disable"
}

func minioEndpoint() string {
	if endpoint := os.Getenv("MINIO_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	return "localhost:9002"
}

func minioAccessKey() string {
	if accessKey := os.Getenv("MINIO_ACCESS_KEY"); accessKey != "" {
		return accessKey
	}
	return "admin"
}

func minioSecretKey() string {
	if secretKey := os.Getenv("MINIO_SECRET_KEY"); secretKey != "" {
		return secretKey
	}
	return "secretpassword"
}

func minioBucket() string {
	if bucket := os.Getenv("MINIO_BUCKET"); bucket != "" {
		return bucket
	}
	return "clips"
}

func whisperURL() string {
	if url := os.Getenv("WHISPER_URL"); url != "" {
		return url
	}
	return "http://localhost:8000/v1/audio/transcriptions"
}
