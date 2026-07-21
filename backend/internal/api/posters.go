package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
	posterflow "github.com/Ripped-sys/StagePoster/backend/internal/poster"
	"github.com/Ripped-sys/StagePoster/backend/internal/repository"
)

func (s *Server) handlePosters(
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

	var payload domain.CreatePosterRequest

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

	ctx, cancel := contextWithTimeout(
		request,
		90*time.Second,
	)
	defer cancel()

	result, err := s.posterFlow.Create(
		ctx,
		payload,
	)

	switch {
	case errors.Is(
		err,
		posterflow.ErrInvalidPosterBrief,
	):
		writeError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return

	case errors.Is(
		err,
		posterflow.ErrUnsupportedStyle,
	):
		writeError(
			writer,
			http.StatusBadRequest,
			err.Error(),
		)
		return

	case err != nil:
		writeError(
			writer,
			http.StatusInternalServerError,
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

func (s *Server) handlePosterRoute(
	writer http.ResponseWriter,
	request *http.Request,
) {
	path := strings.TrimPrefix(
		request.URL.Path,
		"/api/posters/",
	)

	segments := strings.Split(
		strings.Trim(path, "/"),
		"/",
	)

	if len(segments) == 0 ||
		segments[0] == "" {
		writeError(
			writer,
			http.StatusBadRequest,
			"poster id is required",
		)
		return
	}

	posterID := segments[0]

	switch {
	case len(segments) == 1 &&
		request.Method == http.MethodGet:
		s.handlePosterGet(
			writer,
			request,
			posterID,
		)

	case len(segments) == 2 &&
		segments[1] == "result" &&
		request.Method == http.MethodGet:
		s.handlePosterResult(
			writer,
			request,
			posterID,
		)

	case len(segments) == 2 &&
		segments[1] == "select" &&
		request.Method == http.MethodPost:
		s.handlePosterSelect(
			writer,
			request,
			posterID,
		)

	case len(segments) == 4 &&
		segments[1] == "candidates" &&
		segments[3] == "image" &&
		request.Method == http.MethodGet:
		s.handleCandidateImage(
			writer,
			request,
			posterID,
			segments[2],
		)

	default:
		writeError(
			writer,
			http.StatusNotFound,
			"poster route not found",
		)
	}
}

func (s *Server) handlePosterGet(
	writer http.ResponseWriter,
	request *http.Request,
	posterID string,
) {
	ctx, cancel := contextWithTimeout(
		request,
		60*time.Second,
	)
	defer cancel()

	result, err := s.posterFlow.Get(
		ctx,
		posterID,
	)

	if errors.Is(err, repository.ErrNotFound) {
		writeError(
			writer,
			http.StatusNotFound,
			"poster not found",
		)
		return
	}

	if err != nil {
		writeError(
			writer,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handlePosterSelect(
	writer http.ResponseWriter,
	request *http.Request,
	posterID string,
) {
	var payload domain.SelectCandidateRequest

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

	ctx, cancel := contextWithTimeout(
		request,
		30*time.Second,
	)
	defer cancel()

	result, err := s.posterFlow.Select(
		ctx,
		posterID,
		payload.CandidateID,
	)

	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(
			writer,
			http.StatusNotFound,
			"poster or candidate not found",
		)
		return

	case errors.Is(
		err,
		posterflow.ErrCandidateNotReady,
	):
		writeError(
			writer,
			http.StatusConflict,
			err.Error(),
		)
		return

	case err != nil:
		writeError(
			writer,
			http.StatusConflict,
			err.Error(),
		)
		return
	}

	writeJSON(writer, http.StatusOK, result)
}

func (s *Server) handleCandidateImage(
	writer http.ResponseWriter,
	request *http.Request,
	posterID string,
	candidateID string,
) {
	ctx, cancel := contextWithTimeout(
		request,
		60*time.Second,
	)
	defer cancel()

	result, err := s.posterFlow.OpenCandidate(
		ctx,
		posterID,
		candidateID,
	)

	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(
			writer,
			http.StatusNotFound,
			"candidate not found",
		)
		return

	case errors.Is(
		err,
		posterflow.ErrCandidateNotReady,
	):
		writeError(
			writer,
			http.StatusConflict,
			err.Error(),
		)
		return

	case err != nil:
		writeError(
			writer,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	defer result.Body.Close()

	if result.ContentType != "" {
		writer.Header().Set(
			"Content-Type",
			result.ContentType,
		)
	}

	writer.Header().Set(
		"Cache-Control",
		"private, max-age=3600",
	)

	writer.WriteHeader(http.StatusOK)
	_, _ = io.Copy(writer, result.Body)
}

func (s *Server) handlePosterResult(
	writer http.ResponseWriter,
	request *http.Request,
	posterID string,
) {
	ctx, cancel := contextWithTimeout(
		request,
		60*time.Second,
	)
	defer cancel()

	result, err := s.posterFlow.OpenFinalResult(
		ctx,
		posterID,
	)

	switch {
	case errors.Is(
		err,
		posterflow.ErrResultNotReady,
	):
		writeError(
			writer,
			http.StatusConflict,
			err.Error(),
		)
		return

	case errors.Is(err, repository.ErrNotFound):
		writeError(
			writer,
			http.StatusNotFound,
			"poster result not found",
		)
		return

	case err != nil:
		writeError(
			writer,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	defer result.Body.Close()

	writer.Header().Set(
		"Content-Type",
		result.ContentType,
	)
	writer.Header().Set(
		"Cache-Control",
		"private, max-age=3600",
	)

	writer.WriteHeader(http.StatusOK)
	_, _ = io.Copy(writer, result.Body)
}
