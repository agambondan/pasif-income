package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/agambondan/pasif-income/internal/adapters"
	"github.com/agambondan/pasif-income/internal/core/domain"
	"github.com/agambondan/pasif-income/internal/core/ports"
	"github.com/agambondan/pasif-income/internal/services"
	"golang.org/x/oauth2"
)

type generateRequest struct {
	Niche            string               `json:"niche"`
	Topic            string               `json:"topic"`
	Destinations     []domain.Destination `json:"destinations"`
	ScheduleMode     string               `json:"schedule_mode"`
	DripIntervalDays int                  `json:"drip_interval_days"`
	StartAt          string               `json:"start_at"`
}

type apiServer struct {
	repo      ports.Repository
	storage   ports.Storage
	generator *services.GeneratorService
	auth      *services.AuthService
	platform  *services.PlatformService
	metrics   *services.MetricsService
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
		repo:     repo,
		storage:  storage,
		auth:     services.NewAuthService(repo),
		platform: services.NewPlatformService(repo),
		metrics:  services.NewMetricsService(repo),
	}

	generator, err := newGeneratorServiceFromEnv()
	if err != nil {
		log.Fatalf("Generator init failed: %v", err)
	}
	api.generator = generator

	if os.Getenv("GEMINI_API_KEY") == "" {
		log.Println("CRITICAL ERROR: GEMINI_API_KEY is not set. Generation will FAIL.")
		log.Println("Please set it in your .env file or environment variables.")
	}

	// Create default user for testing
	if repo != nil {
		_ = api.auth.Register(context.Background(), "admin", "admin123")
	}

	if repo != nil {
		publisherWorker := services.NewPublisherService(repo, newPublisherFromEnv())
		go publisherWorker.StartWorker(context.Background())
		go api.metrics.StartWorker(context.Background())
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", api.healthHandler)
	mux.HandleFunc("/api/auth/login", api.loginHandler)
	mux.HandleFunc("/api/auth/logout", api.logoutHandler)
	mux.HandleFunc("/api/auth/refresh/", api.refreshTokenHandler)
	mux.HandleFunc("/api/auth/revoke/", api.revokeTokenHandler)
	mux.HandleFunc("/api/platforms", api.platformsHandler)
	mux.HandleFunc("/api/accounts", api.accountsHandler)
	mux.HandleFunc("/api/accounts/", api.deleteAccountHandler)
	mux.HandleFunc("/api/auth/", api.oauthHandler)
	mux.HandleFunc("/api/videos", api.videosHandler)
	mux.HandleFunc("/api/metrics", api.metricsHandler)
	mux.HandleFunc("/api/metrics/sync", api.metricsSyncHandler)
	mux.HandleFunc("/api/clips", api.clipsHandler)
	mux.HandleFunc("/api/generate", api.generateHandler)
	mux.HandleFunc("/api/publish/history", api.publishHistoryHandler)
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

	sessionToken, err := a.auth.CreateSession(r.Context(), user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "cf_session",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((30 * 24 * time.Hour).Seconds()),
	})

	writeJSON(w, http.StatusOK, user)
}

func (a *apiServer) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if token, ok := sessionTokenFromRequest(r); ok && a.repo != nil {
		_ = a.repo.DeleteSession(r.Context(), token)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "cf_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
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
	userID, err := a.currentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	accounts, err := a.repo.ListConnectedAccounts(r.Context(), userID)
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
	userID, err := a.currentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	account, err := a.repo.GetConnectedAccountByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if account.UserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := a.repo.DeleteConnectedAccount(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *apiServer) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	userID, err := a.currentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/auth/refresh/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	acc, err := a.auth.RefreshAccountToken(r.Context(), id, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

func (a *apiServer) revokeTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	userID, err := a.currentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/auth/revoke/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if err := a.auth.RevokeAccountToken(r.Context(), id, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *apiServer) oauthHandler(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/auth/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.NotFound(w, r)
		return
	}
	platform := pathParts[0]
	method := r.URL.Query().Get("method")
	if method == "" {
		method = domain.AuthMethodChromiumProfile
	}
	if method == "browser" {
		method = domain.AuthMethodChromiumProfile
	}
	if method != domain.AuthMethodChromiumProfile && method != domain.AuthMethodAPI {
		http.Error(w, "unsupported auth method", http.StatusBadRequest)
		return
	}

	userID, err := a.currentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if method == domain.AuthMethodChromiumProfile {
		email := strings.TrimSpace(r.URL.Query().Get("email"))
		if email == "" {
			http.Error(w, "email is required for chromium profile linking", http.StatusBadRequest)
			return
		}

		displayName := strings.TrimSpace(r.URL.Query().Get("display_name"))
		if displayName == "" {
			displayName = strings.ToUpper(platform) + " Chromium Profile"
		}

		acc, err := a.auth.LinkConnectedAccount(r.Context(), userID, platform, displayName, email, method, "", "", time.Time{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("Chromium profile ready for %s at %s\n", acc.Email, acc.ProfilePath)

		if len(pathParts) > 1 && pathParts[1] == "callback" {
			writeJSON(w, http.StatusOK, acc)
			return
		}

		http.Redirect(w, r, "http://localhost:13100/integrations", http.StatusFound)
		return
	}

	if platform != "youtube" {
		http.Error(w, "api connect is only wired for youtube", http.StatusNotImplemented)
		return
	}

	stateCookie, _ := r.Cookie("cf_oauth_state")
	if len(pathParts) > 1 && pathParts[1] == "callback" {
		if stateCookie == nil || stateCookie.Value == "" {
			http.Error(w, "oauth state missing", http.StatusBadRequest)
			return
		}
		if got := strings.TrimSpace(r.URL.Query().Get("state")); got == "" || got != stateCookie.Value {
			http.Error(w, "oauth state mismatch", http.StatusBadRequest)
			return
		}
		clearOAuthStateCookie(w)

		acc, err := a.completeYouTubeAPIConnect(r, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, acc)
		return
	}

	state, err := randomOAuthState()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "cf_oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((10 * time.Minute).Seconds()),
	})

	authURL, err := youtubeAuthURL(state)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (a *apiServer) completeYouTubeAPIConnect(r *http.Request, userID int) (*domain.ConnectedAccount, error) {
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		return nil, errors.New("oauth code is required")
	}

	cfg, err := youtubeOAuthConfig()
	if err != nil {
		return nil, err
	}

	token, err := cfg.Exchange(r.Context(), code)
	if err != nil {
		return nil, fmt.Errorf("exchange oauth code: %w", err)
	}

	email, displayName, err := fetchGoogleUserInfo(r.Context(), token.AccessToken)
	if err != nil {
		return nil, err
	}
	if displayName == "" {
		displayName = "YouTube API"
	}

	return a.auth.LinkConnectedAccount(r.Context(), userID, "youtube", displayName, email, domain.AuthMethodAPI, token.AccessToken, token.RefreshToken, token.Expiry)
}

func youtubeOAuthConfig() (*oauth2.Config, error) {
	clientID := strings.TrimSpace(os.Getenv("YOUTUBE_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("YOUTUBE_CLIENT_SECRET"))
	redirectURL := strings.TrimSpace(os.Getenv("YOUTUBE_REDIRECT_URL"))
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, errors.New("youtube oauth config is incomplete; set YOUTUBE_CLIENT_ID, YOUTUBE_CLIENT_SECRET, and YOUTUBE_REDIRECT_URL")
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       youtubeOAuthScopes(),
		Endpoint:     oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token"},
	}, nil
}

func youtubeOAuthScopes() []string {
	raw := strings.TrimSpace(os.Getenv("YOUTUBE_SCOPES"))
	if raw == "" {
		return []string{
			"https://www.googleapis.com/auth/youtube.upload",
			"https://www.googleapis.com/auth/youtube.readonly",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"openid",
		}
	}

	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' '
	})
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			scopes = append(scopes, trimmed)
		}
	}
	return scopes
}

func fetchGoogleUserInfo(ctx context.Context, accessToken string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("userinfo lookup failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Email       string `json:"email"`
		Name        string `json:"name"`
		GivenName   string `json:"given_name"`
		Verified    bool   `json:"verified_email"`
		ProfileLink string `json:"profile"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", "", err
	}

	displayName := strings.TrimSpace(payload.Name)
	if displayName == "" {
		displayName = strings.TrimSpace(payload.GivenName)
	}
	return strings.TrimSpace(payload.Email), displayName, nil
}

func youtubeAuthURL(state string) (string, error) {
	cfg, err := youtubeOAuthConfig()
	if err != nil {
		return "", err
	}
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce, oauth2.SetAuthURLParam("include_granted_scopes", "true")), nil
}

func randomOAuthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func clearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "cf_oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
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

type metricsSummary struct {
	TotalVideos       int        `json:"total_videos"`
	TotalViews        uint64     `json:"total_views"`
	TotalLikes        uint64     `json:"total_likes"`
	TotalComments     uint64     `json:"total_comments"`
	LatestCollectedAt *time.Time `json:"latest_collected_at,omitempty"`
}

type metricsResponse struct {
	Summary metricsSummary               `json:"summary"`
	Latest  []domain.VideoMetricSnapshot `json:"latest"`
	History []domain.VideoMetricSnapshot `json:"history"`
}

func (a *apiServer) metricsHandler(w http.ResponseWriter, r *http.Request) {
	if a.repo == nil {
		http.Error(w, "repository unavailable", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userID, err := a.currentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	history, err := a.repo.ListVideoMetricSnapshots(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	latest := collapseLatestMetricSnapshots(history)
	summary := summarizeMetrics(latest)
	writeJSON(w, http.StatusOK, metricsResponse{
		Summary: summary,
		Latest:  latest,
		History: history,
	})
}

func (a *apiServer) metricsSyncHandler(w http.ResponseWriter, r *http.Request) {
	if a.repo == nil || a.metrics == nil {
		http.Error(w, "metrics unavailable", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userID, err := a.currentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	synced, err := a.metrics.SyncUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"synced": synced,
	})
}

func collapseLatestMetricSnapshots(history []domain.VideoMetricSnapshot) []domain.VideoMetricSnapshot {
	seen := make(map[string]struct{})
	latest := make([]domain.VideoMetricSnapshot, 0, len(history))
	for _, snap := range history {
		if _, ok := seen[snap.ExternalID]; ok {
			continue
		}
		seen[snap.ExternalID] = struct{}{}
		latest = append(latest, snap)
	}
	return latest
}

func summarizeMetrics(latest []domain.VideoMetricSnapshot) metricsSummary {
	var summary metricsSummary
	summary.TotalVideos = len(latest)
	for i, snap := range latest {
		summary.TotalViews += snap.ViewCount
		summary.TotalLikes += snap.LikeCount
		summary.TotalComments += snap.CommentCount
		if i == 0 {
			collected := snap.CollectedAt
			summary.LatestCollectedAt = &collected
			continue
		}
		if summary.LatestCollectedAt == nil || snap.CollectedAt.After(*summary.LatestCollectedAt) {
			collected := snap.CollectedAt
			summary.LatestCollectedAt = &collected
		}
	}
	return summary
}

func newGeneratorServiceFromEnv() (*services.GeneratorService, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	accessToken := os.Getenv("GEMINI_ACCESS_TOKEN")
	if apiKey == "" && accessToken == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY or GEMINI_ACCESS_TOKEN must be set")
	}

	writer := adapters.NewGeminiWriter(apiKey)
	voice := adapters.NewVoiceAdapter("en-US-Standard-A")
	image := adapters.NewStableDiffusionAdapter(os.Getenv("SD_API_URL"))
	assembler := adapters.NewFFmpegAssembler()
	uploader, err := newUploaderFromEnv()
	if err != nil {
		return nil, err
	}

	branding := services.NewBrandingService(image)
	qc := services.NewQualityControlService(apiKey)
	return services.NewGeneratorService(writer, voice, image, assembler, uploader, branding, qc), nil
}

func newUploaderFromEnv() (ports.Uploader, error) {
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
		return nil, fmt.Errorf("init minio uploader: %w", err)
	}

	return uploader, nil
}

func newPublisherFromEnv() ports.Publisher {
	return adapters.NewDistributionPublisher()
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

		userID, err := a.currentUserID(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
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

		go a.runGeneration(userID, job.ID, niche, topic, req.Destinations, req.ScheduleMode, req.DripIntervalDays, req.StartAt)

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

	if strings.HasSuffix(id, "/distributions") {
		jobID := strings.TrimSuffix(id, "/distributions")
		jobID = strings.TrimSuffix(jobID, "/")
		if jobID == "" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		a.distributionsHandler(w, r, jobID)
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

func (a *apiServer) distributionsHandler(w http.ResponseWriter, r *http.Request, generationJobID string) {
	if a.repo == nil {
		http.Error(w, "repository unavailable", http.StatusServiceUnavailable)
		return
	}

	jobs, err := a.repo.ListDistributionJobs(r.Context(), generationJobID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, jobs)
}

func (a *apiServer) publishHistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if a.repo == nil {
		http.Error(w, "repository unavailable", http.StatusServiceUnavailable)
		return
	}
	userID, err := a.currentUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	jobs, err := a.repo.ListAllDistributionJobs(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (a *apiServer) runGeneration(userID int, jobID, niche, topic string, destinations []domain.Destination, scheduleMode string, dripIntervalDays int, startAt string) {
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

	story, err := a.generator.GenerateContent(ctx, niche, topic)
	if err != nil {
		if a.repo != nil {
			_ = a.repo.UpdateJobStatus(ctx, jobID, "failed", err.Error())
		}
		return
	}

	description := fmt.Sprintf("#%s #%s #ai #faceless", niche, strings.ReplaceAll(topic, " ", ""))

	if a.repo != nil {
		if err := a.repo.UpdateJobArtifact(ctx, jobID, story.Title, description, story.VideoOutput); err != nil {
			log.Printf("failed to store job artifact: %v\n", err)
		}
	}

	// Create distribution jobs if successful
	if a.repo != nil {
		for i, dest := range destinations {
			account, err := a.repo.GetConnectedAccountByID(ctx, dest.AccountID)
			if err != nil {
				log.Printf("failed to load account %s: %v\n", dest.AccountID, err)
				continue
			}
			if account.UserID != userID {
				log.Printf("skipping account %s because it does not belong to current user", account.ID)
				continue
			}
			if account.PlatformID != dest.Platform {
				log.Printf("destination platform mismatch for account %s", account.ID)
				continue
			}

			distJob := domain.DistributionJob{
				GenerationJobID: jobID,
				AccountID:       dest.AccountID,
				Platform:        dest.Platform,
				Status:          "pending",
				StatusDetail:    "queued",
				ScheduledAt:     computeScheduledAt(scheduleMode, dripIntervalDays, startAt, i),
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

func computeScheduledAt(scheduleMode string, dripIntervalDays int, startAt string, index int) *time.Time {
	mode := strings.TrimSpace(strings.ToLower(scheduleMode))
	if mode == "" || mode == "immediate" {
		return nil
	}

	loc := schedulingLocation()
	now := time.Now().In(loc)
	base := now
	if parsed, err := parseScheduleTime(startAt, loc); err == nil {
		base = parsed
	} else {
		base = nextPrimeTime(now, loc)
	}

	if dripIntervalDays < 1 {
		dripIntervalDays = 1
	}

	switch mode {
	case "prime_time":
		scheduled := base.AddDate(0, 0, index)
		return &scheduled
	case "drip_feed":
		scheduled := base.AddDate(0, 0, index*dripIntervalDays)
		return &scheduled
	default:
		return nil
	}
}

func parseScheduleTime(raw string, loc *time.Location) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty schedule time")
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.In(loc), nil
	}
	if parsed, err := time.ParseInLocation("2006-01-02 15:04", raw, loc); err == nil {
		return parsed, nil
	}
	return time.Time{}, fmt.Errorf("invalid schedule time")
}

func nextPrimeTime(now time.Time, loc *time.Location) time.Time {
	hour := 19
	if raw := strings.TrimSpace(os.Getenv("PRIME_UPLOAD_HOUR")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 && parsed <= 23 {
			hour = parsed
		}
	}

	candidate := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, loc)
	if !candidate.After(now) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate
}

func schedulingLocation() *time.Location {
	tz := strings.TrimSpace(os.Getenv("SCHEDULING_TIMEZONE"))
	if tz == "" {
		return time.Local
	}
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	return time.Local
}

func (a *apiServer) currentUserID(r *http.Request) (int, error) {
	if a.repo == nil {
		return 0, errors.New("repository unavailable")
	}
	token, ok := sessionTokenFromRequest(r)
	if !ok {
		return 0, errors.New("unauthorized")
	}
	user, err := a.repo.GetUserBySessionToken(r.Context(), token)
	if err != nil {
		return 0, errors.New("unauthorized")
	}
	return user.ID, nil
}

func sessionTokenFromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie("cf_session")
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
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
