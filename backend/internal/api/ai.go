package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
	posterflow "github.com/Ripped-sys/StagePoster/backend/internal/poster"
	"github.com/Ripped-sys/StagePoster/backend/internal/repository"
)

func (s *Server) handleAIDesign(
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

	if !s.aiReady() {
		writeError(
			writer,
			http.StatusServiceUnavailable,
			"AI service is not configured",
		)
		return
	}

	request.Body = http.MaxBytesReader(
		writer,
		request.Body,
		1024*1024,
	)

	var payload domain.AIDesignRequest

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

	payload.Event.Title = strings.TrimSpace(
		payload.Event.Title,
	)

	if payload.Event.Title == "" {
		writeError(
			writer,
			http.StatusBadRequest,
			"event.title is required",
		)
		return
	}

	ctx, cancel := contextWithTimeout(
		request,
		4*time.Minute,
	)
	defer cancel()

	release, err := s.aiRuntime.Acquire(ctx)
	if err != nil {
		writeError(
			writer,
			http.StatusBadGateway,
			err.Error(),
		)
		return
	}
	defer release()

	result, metrics, err := s.aiService.Plan(
		ctx,
		payload.Event,
		payload.Visual,
		payload.Message,
	)
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
		http.StatusOK,
		domain.AIDesignResponse{
			Result: result,
			Metrics: domain.AIMetricsResponse{
				LatencyMS: metrics.Latency.Milliseconds(),
				PromptTokens: metrics.PromptTokens,
				CompletionTokens: metrics.CompletionTokens,
			},
		},
	)
}

func (s *Server) handlePosterReview(
	writer http.ResponseWriter,
	request *http.Request,
	posterID string,
) {
	if !s.aiReady() {
		writeError(
			writer,
			http.StatusServiceUnavailable,
			"AI service is not configured",
		)
		return
	}

	request.Body = http.MaxBytesReader(
		writer,
		request.Body,
		1024*1024,
	)

	var payload domain.PosterReviewRequest

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&payload); err != nil &&
		!errors.Is(err, io.EOF) {
		writeError(
			writer,
			http.StatusBadRequest,
			"invalid JSON: "+err.Error(),
		)
		return
	}

	ctx, cancel := contextWithTimeout(
		request,
		5*time.Minute,
	)
	defer cancel()

	material, err := s.posterFlow.ReviewMaterial(
		ctx,
		posterID,
	)

	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(
			writer,
			http.StatusNotFound,
			"poster not found",
		)
		return

	case errors.Is(
		err,
		posterflow.ErrResultNotReady,
	):
		writeError(
			writer,
			http.StatusConflict,
			"final poster is not ready",
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

	var event domain.EventBrief

	if err := json.Unmarshal(
		[]byte(material.Poster.EventJSON),
		&event,
	); err != nil {
		writeError(
			writer,
			http.StatusInternalServerError,
			"decode persisted event brief: "+err.Error(),
		)
		return
	}

	var visual domain.VisualBrief

	if err := json.Unmarshal(
		[]byte(material.Poster.VisualJSON),
		&visual,
	); err != nil {
		writeError(
			writer,
			http.StatusInternalServerError,
			"decode persisted visual brief: "+err.Error(),
		)
		return
	}

	release, err := s.aiRuntime.Acquire(ctx)
	if err != nil {
		writeError(
			writer,
			http.StatusBadGateway,
			err.Error(),
		)
		return
	}
	defer release()

	result, metrics, err := s.aiService.Review(
		ctx,
		material.Output.StoragePath,
		event,
		visual,
		payload.DesignPlan,
	)
	if err != nil {
		writeError(
			writer,
			http.StatusBadGateway,
			err.Error(),
		)
		return
	}

	round, err := s.posterFlow.NextReviewRound(
		ctx,
		posterID,
	)
	if err != nil {
		writeError(
			writer,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	reviewID, err := domain.NewID("review_")
	if err != nil {
		writeError(
			writer,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	candidateID := material.Output.CandidateID

	if candidateID == "" &&
		material.Candidate != nil {
		candidateID = material.Candidate.ID
	}

	review := domain.PosterReviewRecord{
		ID:          reviewID,
		PosterID:    posterID,
		OutputID:    material.Output.ID,
		CandidateID: candidateID,

		Round:      round,
		TotalScore: result.TotalScore,
		Decision:   result.Decision,
		Result:     result,

		Model:            s.aiModel,
		PromptTokens:     metrics.PromptTokens,
		CompletionTokens: metrics.CompletionTokens,
		LatencyMS:        metrics.Latency.Milliseconds(),

		CreatedAt: time.Now().UTC(),
	}

	if err := s.posterFlow.SaveReview(
		ctx,
		review,
	); err != nil {
		writeError(
			writer,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(
		writer,
		http.StatusCreated,
		domain.PosterReviewResponse{
			Review: review,
		},
	)
}

func (s *Server) handlePosterReviews(
	writer http.ResponseWriter,
	request *http.Request,
	posterID string,
) {
	limit := 20

	if raw := request.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(
				writer,
				http.StatusBadRequest,
				"invalid limit",
			)
			return
		}

		limit = parsed
	}

	ctx, cancel := contextWithTimeout(
		request,
		20*time.Second,
	)
	defer cancel()

	reviews, err := s.posterFlow.ListReviews(
		ctx,
		posterID,
		limit,
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

	writeJSON(
		writer,
		http.StatusOK,
		domain.PosterReviewListResponse{
			PosterID: posterID,
			Reviews:  reviews,
		},
	)
}

func (s *Server) handlePosterTimeline(
	writer http.ResponseWriter,
	request *http.Request,
	posterID string,
) {
	ctx, cancel := contextWithTimeout(
		request,
		60*time.Second,
	)
	defer cancel()

	posterResult, err := s.posterFlow.Get(
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

	reviews, err := s.posterFlow.ListReviews(
		ctx,
		posterID,
		100,
	)
	if err != nil {
		writeError(
			writer,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(
		writer,
		http.StatusOK,
		map[string]any{
			"posterId": posterID,
			"poster":   posterResult,
			"reviews":  reviews,
		},
	)
}

func (s *Server) handleDependencies(
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

	ctx, cancel := contextWithTimeout(
		request,
		15*time.Second,
	)
	defer cancel()

	statusCode := http.StatusOK
	overall := "healthy"

	database := map[string]any{
		"status": "ready",
	}

	if err := s.service.DatabaseHealth(ctx); err != nil {
		database["status"] = "unavailable"
		database["error"] = err.Error()

		statusCode = http.StatusServiceUnavailable
		overall = "degraded"
	}

	comfy := map[string]any{
		"status": "ready",
	}

	if err := s.service.ComfyHealth(ctx); err != nil {
		comfy["status"] = "unavailable"
		comfy["error"] = err.Error()

		statusCode = http.StatusServiceUnavailable
		overall = "degraded"
	}

	vlm := map[string]any{
		"url":   s.aiURL,
		"model": s.aiModel,
	}

	if !s.aiReady() {
		vlm["status"] = "disabled"

		statusCode = http.StatusServiceUnavailable
		overall = "degraded"
	} else if err := s.aiClient.Health(ctx); err != nil {
		vlm["status"] = "unavailable"
		vlm["error"] = err.Error()

		statusCode = http.StatusServiceUnavailable
		overall = "degraded"
	} else {
		vlm["status"] = "ready"

		sleeping, sleepErr :=
			s.aiClient.IsSleeping(ctx)

		if sleepErr == nil {
			vlm["sleeping"] = sleeping
		} else {
			vlm["sleepState"] = "unknown"
			vlm["sleepError"] = sleepErr.Error()
		}
	}

	writeJSON(
		writer,
		statusCode,
		map[string]any{
			"status": overall,
			"dependencies": map[string]any{
				"database": database,
				"comfyui": comfy,
				"vlm":     vlm,
			},
			"tokenRequired": s.apiToken != "",
		},
	)
}

func (s *Server) aiReady() bool {
	return s.aiClient != nil &&
		s.aiService != nil &&
		s.aiRuntime != nil
}
