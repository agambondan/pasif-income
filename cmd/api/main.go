package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/adapters"
	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/agambondan/pasif-income/internal/core/ports"
	"github.com/agambondan/pasif-income/internal/services"
)

type generateRequest struct {
	Niche        string               `json:"niche"`
	Topic        string               `json:"topic"`
	Destinations []domain.Destination `json:"destinations"`
}

type apiServer struct {
	repo      ports.Repository
	storage   ports.Storage
	generator *services.GeneratorService
	auth      *services.AuthService
	platform  *services.PlatformService
}

func main() {
	log.Println("--- Dashboard API Starting ---")

	var repo ports.Repository
	if dbRepo, err := adapters.NewPostgresRepository(postgresDSN()); err != nil {
		log.Printf("Postgres Warning: %v\n", err)
	} else {
		repo = dbRepo
	}

	storage, err := adapters.NewMinIOStorage(minioEndpoint(), minioAccessKey(), minioSecretKey(), minioBucket())
	if err != nil {
		log.Printf("MinIO Warning: %v\n", err)
	}

	api := &apiServer{
		repo:      repo,
		storage:   storage,
		generator: newGeneratorServiceFromEnv(),
		auth:      services.NewAuthService(repo),
		platform:  services.NewPlatformService(repo),
	}

	// Create default user for testing
	if repo != nil {
		_ = api.auth.Register(context.Background(), "admin", "admin123")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", api.healthHandler)
	mux.HandleFunc("/api/auth/login", api.loginHandler)
	mux.HandleFunc("/api/platforms", api.platformsHandler)
	mux.HandleFunc("/api/accounts", api.accountsHandler)
	mux.HandleFunc("/api/accounts/", api.deleteAccountHandler)
	mux.HandleFunc("/api/auth/", api.oauthHandler)
	mux.HandleFunc("/api/videos", api.videosHandler)
	mux.HandleFunc("/api/clips", api.clipsHandler)
	mux.HandleFunc("/api/generate", api.generateHandler)
	mux.HandleFunc("/api/jobs", api.jobsHandler)
	mux.HandleFunc("/api/jobs/", api.jobByIDHandler)

	handler := withCORS(mux)

	log.Println("Listening on :8080 for dashboard...")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func minioEndpoint() string {
	if e := os.Getenv("MINIO_ENDPOINT"); e != "" {
		return e
	}
	return "localhost:9002"
}

func minioAccessKey() string {
	if k := os.Getenv("MINIO_ACCESS_KEY"); k != "" {
		return k
	}
	return "admin"
}

func minioSecretKey() string {
	if s := os.Getenv("MINIO_SECRET_KEY"); s != "" {
		return s
	}
	return "secretpassword"
}

func minioBucket() string {
	if b := os.Getenv("MINIO_BUCKET"); b != "" {
		return b
	}
	return "clips"
}

func (a *apiServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := a.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (a *apiServer) platformsHandler(w http.ResponseWriter, r *http.Request) {
	platforms := a.platform.GetSupportedPlatforms()
	writeJSON(w, http.StatusOK, platforms)
}

func (a *apiServer) accountsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// Stubbing user_id=1 for now as there's no session middleware yet.
	accounts, err := a.repo.ListConnectedAccounts(r.Context(), 1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, accounts)
}

func (a *apiServer) deleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if err := a.repo.DeleteConnectedAccount(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *apiServer) oauthHandler(w http.ResponseWriter, r *http.Request) {
	// Simple stub for OAuth flow
	// /api/auth/{platform} -> redirect to provider (mock)
	// /api/auth/{platform}/callback -> save token
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/auth/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.NotFound(w, r)
		return
	}
	platform := pathParts[0]

	if len(pathParts) > 1 && pathParts[1] == "callback" {
		// Mock callback: just save a dummy connected account
		acc := &domain.ConnectedAccount{
			ID:          platform + "-test-" + makeJobID()[:8],
			UserID:      1,
			PlatformID:  platform,
			DisplayName: "Test " + strings.ToUpper(platform) + " Account",
			AccessToken: "dummy-token",
			Expiry:      time.Now().Add(365 * 24 * time.Hour),
			CreatedAt:   time.Now(),
		}
		if err := a.repo.SaveConnectedAccount(r.Context(), acc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Redirect back to frontend integrations page
		http.Redirect(w, r, "http://localhost:13100/integrations", http.StatusFound)
		return
	}

	// Mock redirect
	http.Redirect(w, r, "/api/auth/"+platform+"/callback?code=mock", http.StatusFound)
}

func (a *apiServer) videosHandler(w http.ResponseWriter, r *http.Request) {
	if a.storage == nil {
		http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
		return
	}
	files, err := a.storage.ListFiles(r.Context(), "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, files)
}

func newGeneratorServiceFromEnv() *services.GeneratorService {
	apiKey := os.Getenv("GEMINI_API_KEY")

	writer := adapters.NewGeminiWriter(apiKey)
	voice := adapters.NewVoiceAdapter("en-US-Standard-A")
	image := adapters.NewStableDiffusionAdapter(os.Getenv("SD_API_URL"))
	assembler := adapters.NewFFmpegAssembler()
	uploader := newUploaderFromEnv()

	return services.NewGeneratorService(writer, voice, image, assembler, uploader)
}

func newUploaderFromEnv() ports.Uploader {
	if os.Getenv("USE_MOCK") == "true" {
		return adapters.NewMockUploader("YouTube Shorts")
	}

	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9002"
	}
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "admin"
	}
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "secretpassword"
	}
	bucket := os.Getenv("MINIO_BUCKET")
	if bucket == "" {
		bucket = "clips"
	}

	uploader, err := adapters.NewMinIOUploader(endpoint, accessKey, secretKey, bucket, "YouTube Shorts")
	if err != nil {
		log.Printf("MinIO uploader warning: %v (falling back to mock)\n", err)
		return adapters.NewMockUploader("YouTube Shorts")
	}

	return uploader
}

func postgresDSN() string {
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return dsn
	}
	return "postgres://factory:secretpassword@localhost:5432/clips_db?sslmode=disable"
}

func (a *apiServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (a *apiServer) clipsHandler(w http.ResponseWriter, r *http.Request) {
	if a.repo == nil {
		http.Error(w, "repository unavailable", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		clips, err := a.repo.ListClips(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, clips)
	case http.MethodPatch:
		var update struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if update.ID == "" || update.Status == "" {
			http.Error(w, "id and status are required", http.StatusBadRequest)
			return
		}
		if err := a.repo.UpdateStatus(r.Context(), update.ID, update.Status); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *apiServer) generateHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req generateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		niche := strings.TrimSpace(req.Niche)
		if niche == "" {
			niche = os.Getenv("NICHE")
		}
		if niche == "" {
			niche = "stoicism"
		}

		topic := strings.TrimSpace(req.Topic)
		if topic == "" {
			topic = os.Getenv("TOPIC")
		}
		if topic == "" {
			topic = "how to control your mind"
		}

		job := domain.GenerationJob{
			ID:        makeJobID(),
			Niche:     niche,
			Topic:     topic,
			Status:    "queued",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if a.repo == nil {
			http.Error(w, "repository unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := a.repo.CreateJob(r.Context(), &job); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		go a.runGeneration(job.ID, niche, topic, req.Destinations)

		writeJSON(w, http.StatusAccepted, job)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *apiServer) jobsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if a.repo == nil {
			http.Error(w, "repository unavailable", http.StatusServiceUnavailable)
			return
		}
		jobs, err := a.repo.ListJobs(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, jobs)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *apiServer) jobByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	if strings.HasSuffix(id, "/complete") {
		jobID := strings.TrimSuffix(id, "/complete")
		jobID = strings.TrimSuffix(jobID, "/")
		if jobID == "" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost && r.Method != http.MethodPatch {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		a.updateJobStatusHandler(w, r, jobID, true)
		return
	}

	if a.repo == nil {
		http.Error(w, "repository unavailable", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		job, err := a.repo.GetJob(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, job)
	case http.MethodPatch:
		a.updateJobStatusHandler(w, r, id, false)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *apiServer) updateJobStatusHandler(w http.ResponseWriter, r *http.Request, id string, completeDefault bool) {
	if a.repo == nil {
		http.Error(w, "repository unavailable", http.StatusServiceUnavailable)
		return
	}

	var update struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if completeDefault && update.Status == "" {
		update.Status = "completed"
	}
	if update.Status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}
	if err := a.repo.UpdateJobStatus(r.Context(), id, update.Status, update.Error); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	job, err := a.repo.GetJob(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (a *apiServer) runGeneration(jobID, niche, topic string, destinations []domain.Destination) {
	if a.repo != nil {
		if err := a.repo.UpdateJobStatus(context.Background(), jobID, "running", ""); err != nil {
			log.Printf("failed to mark job running: %v\n", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	if a.generator == nil {
		if a.repo != nil {
			_ = a.repo.UpdateJobStatus(ctx, jobID, "failed", "generator unavailable")
		}
		return
	}

	if err := a.generator.GenerateContent(ctx, niche, topic); err != nil {
		if a.repo != nil {
			_ = a.repo.UpdateJobStatus(ctx, jobID, "failed", err.Error())
		}
		return
	}

	// Create distribution jobs if successful
	if a.repo != nil {
		for _, dest := range destinations {
			distJob := domain.DistributionJob{
				GenerationJobID: jobID,
				AccountID:       dest.AccountID,
				Platform:        dest.Platform,
				Status:          "pending",
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			if err := a.repo.CreateDistributionJob(ctx, &distJob); err != nil {
				log.Printf("failed to create distribution job for %s: %v\n", dest.Platform, err)
			}
		}

		if err := a.repo.UpdateJobStatus(ctx, jobID, "completed", ""); err != nil {
			log.Printf("failed to mark job completed: %v\n", err)
		}
	}
}

func makeJobID() string {
	return time.Now().UTC().Format("20060102150405.000000000")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
