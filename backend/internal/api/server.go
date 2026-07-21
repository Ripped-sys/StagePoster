package api

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
	"github.com/Ripped-sys/StagePoster/backend/internal/service"
)

type Server struct {
	service    *service.PosterService
	apiToken   string
	corsOrigin string
}

func NewServer(
	posterService *service.PosterService,
	apiToken string,
	corsOrigin string,
) *Server {
	if corsOrigin == "" {
		corsOrigin = "*"
	}

	return &Server{
		service:    posterService,
		apiToken:   apiToken,
		corsOrigin: corsOrigin,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/generate", s.handleGenerate)
	mux.HandleFunc("/api/jobs/", s.handleJobs)

	return s.cors(s.auth(mux))
}

func (s *Server) handleHealth(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodGet {
		writeError(
			writer,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
		return
	}

	ctx, cancel := contextWithTimeout(request, 10*time.Second)
	defer cancel()

	if err := s.service.Health(ctx); err != nil {
		writeJSON(
			writer,
			http.StatusServiceUnavailable,
			map[string]any{
				"status": "degraded",
				"error":  err.Error(),
			},
		)
		return
	}

	writeJSON(
		writer,
		http.StatusOK,
		map[string]any{
			"status":        "ok",
			"comfy":         "connected",
			"bindings":      s.service.Bindings(),
			"tokenRequired": s.apiToken != "",
		},
	)
}

func (s *Server) handleGenerate(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodPost {
		writeError(
			writer,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
		return
	}

	request.Body = http.MaxBytesReader(
		writer,
		request.Body,
		1024*1024,
	)

	var payload domain.GenerateRequest

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&payload); err != nil {
		writeError(
			writer,
			http.StatusBadRequest,
			"invalid JSON: "+err.Error(),
		)
		return
	}

	ctx, cancel := contextWithTimeout(request, 70*time.Second)
	defer cancel()

	result, err := s.service.Generate(ctx, payload)
	if err != nil {
		writeError(
			writer,
			http.StatusBadGateway,
			err.Error(),
		)
		return
	}

	writeJSON(
		writer,
		http.StatusAccepted,
		result,
	)
}

func (s *Server) handleJobs(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodGet {
		writeError(
			writer,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
		return
	}

	path := strings.TrimPrefix(
		request.URL.Path,
		"/api/jobs/",
	)
	path = strings.Trim(path, "/")

	if path == "" {
		writeError(
			writer,
			http.StatusBadRequest,
			"job id is required",
		)
		return
	}

	if strings.HasSuffix(path, "/result") {
		jobID := strings.TrimSuffix(path, "/result")
		jobID = strings.Trim(jobID, "/")

		s.handleResult(writer, request, jobID)
		return
	}

	s.handleStatus(writer, request, path)
}

func (s *Server) handleStatus(
	writer http.ResponseWriter,
	request *http.Request,
	jobID string,
) {
	ctx, cancel := contextWithTimeout(request, 20*time.Second)
	defer cancel()

	result, err := s.service.Status(ctx, jobID)
	if err != nil {
		writeError(
			writer,
			http.StatusBadGateway,
			err.Error(),
		)
		return
	}

	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleResult(
	writer http.ResponseWriter,
	request *http.Request,
	jobID string,
) {
	ctx, cancel := contextWithTimeout(request, 60*time.Second)
	defer cancel()

	response, err := s.service.OpenResult(ctx, jobID)
	if err != nil {
		writeError(
			writer,
			http.StatusConflict,
			err.Error(),
		)
		return
	}
	defer response.Body.Close()

	if contentType := response.Header.Get("Content-Type"); contentType != "" {
		writer.Header().Set("Content-Type", contentType)
	}

	writer.Header().Set(
		"Cache-Control",
		"private, max-age=3600",
	)

	writer.WriteHeader(http.StatusOK)
	_, _ = io.Copy(writer, response.Body)
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			if s.apiToken == "" ||
				!strings.HasPrefix(request.URL.Path, "/api/") {
				next.ServeHTTP(writer, request)
				return
			}

			token := request.Header.Get("X-Poster-Token")

			if token == "" {
				authorization := request.Header.Get("Authorization")
				token = strings.TrimPrefix(
					authorization,
					"Bearer ",
				)
			}

			if token == "" {
				token = request.URL.Query().Get("token")
			}

			valid := subtle.ConstantTimeCompare(
				[]byte(token),
				[]byte(s.apiToken),
			) == 1

			if !valid {
				writeError(
					writer,
					http.StatusUnauthorized,
					"unauthorized",
				)
				return
			}

			next.ServeHTTP(writer, request)
		},
	)
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set(
				"Access-Control-Allow-Origin",
				s.corsOrigin,
			)
			writer.Header().Set(
				"Access-Control-Allow-Headers",
				"Content-Type, Authorization, X-Poster-Token",
			)
			writer.Header().Set(
				"Access-Control-Allow-Methods",
				"GET, POST, OPTIONS",
			)

			if request.Method == http.MethodOptions {
				writer.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(writer, request)
		},
	)
}

func writeJSON(
	writer http.ResponseWriter,
	status int,
	value any,
) {
	writer.Header().Set(
		"Content-Type",
		"application/json",
	)
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(
	writer http.ResponseWriter,
	status int,
	message string,
) {
	writeJSON(
		writer,
		status,
		map[string]any{
			"error": message,
		},
	)
}
