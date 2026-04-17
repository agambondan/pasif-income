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
		transcriber = adapters.NewWhisperTranscriber("http://localhost:8000/v1/audio/transcriptions")
		agent = adapters.NewGeminiAgent(apiKey)
	}

	vision := adapters.NewPythonVisionAgent("scripts/face_tracker.py")

	editor := adapters.NewFFmpegEditor()

	// Storage (MinIO)
	storage, err := adapters.NewMinIOStorage("localhost:9002", "admin", "secretpassword", "clips")
	if err != nil {
		log.Printf("MinIO Warning: %v (Continuing...)\n", err)
	}

	// Repository (Postgres)
	repo, err := adapters.NewPostgresRepository("postgres://factory:secretpassword@localhost:5432/clips_db?sslmode=disable")
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

	log.Println("Listening on :8080 for dashboard...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
