package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
				LatencyMS:        metrics.Latency.Milliseconds(),
				PromptTokens:     metrics.PromptTokens,
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
		writeInternalError(
			writer,
			request,
			err,
		)
		return
	}

	var event domain.EventBrief

	if err := json.Unmarshal(
		[]byte(material.Poster.EventJSON),
		&event,
	); err != nil {
		writeInternalError(
			writer,
			request,
			fmt.Errorf("decode persisted event brief: %w", err),
		)
		return
	}

	var visual domain.VisualBrief

	if err := json.Unmarshal(
		[]byte(material.Poster.VisualJSON),
		&visual,
	); err != nil {
		writeInternalError(
			writer,
			request,
			fmt.Errorf("decode persisted visual brief: %w", err),
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
		writeInternalError(
			writer,
			request,
			err,
		)
		return
	}

	reviewID, err := domain.NewID("review_")
	if err != nil {
		writeInternalError(
			writer,
			request,
			err,
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
		writeInternalError(
			writer,
			request,
			err,
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
	page, ok := parsePage(writer, request)
	if !ok {
		return
	}

	ctx, cancel := contextWithTimeout(
		request,
		20*time.Second,
	)
	defer cancel()

	reviews, total, err := s.posterFlow.ListReviews(
		ctx,
		posterID,
		page,
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
		writeInternalError(
			writer,
			request,
			err,
		)
		return
	}

	writeJSON(
		writer,
		http.StatusOK,
		domain.PosterReviewListResponse{
			PosterID: posterID,
			Reviews:  reviews,
			ListMeta: domain.NewListMeta(
				page,
				len(reviews),
				total,
			),
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
		writeInternalError(
			writer,
			request,
			err,
		)
		return
	}

	reviews, _, err := s.posterFlow.ListReviews(
		ctx,
		posterID,
		domain.NormalizePage(
			domain.MaxPageLimit,
			0,
		),
	)
	if err != nil {
		writeInternalError(
			writer,
			request,
			err,
		)
		return
	}

	// 任务级成本汇总。每轮的 token 和耗时一直在写库，之前没有任何地方加总。
	metrics, err := s.posterFlow.Metrics(
		ctx,
		posterID,
	)
	if err != nil {
		writeInternalError(
			writer,
			request,
			err,
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
			"metrics":  metrics,
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
				"comfyui":  comfy,
				"vlm":      vlm,
			},
			"capabilities":  s.capabilities(),
			"tokenRequired": s.apiToken != "",
		},
	)
}

// capabilities 如实报告哪些能力真的接通了。
//
// 之前无从判断：素材上传成功、字段返回了、actuallyUsed 是 false，但看不出这是
// "这次没用上"还是"整条链路根本不存在"。前端据此决定要不要显示对应入口，
// 而不是靠猜。
func (s *Server) capabilities() map[string]any {
	negative := map[string]any{
		"available": false,
		"reason": "no negative text node bound, " +
			"or sampler cfg is 1",
	}

	if s.service != nil {
		if effective, cfg, node := s.service.NegativePromptState(); effective {
			negative = map[string]any{
				"available": true,
				"node":      node,
				"cfg":       cfg,
			}
		} else {
			negative["cfg"] = cfg
		}
	}

	return map[string]any{
		"negativePrompt": negative,

		// 参考图只影响需求理解那次 VLM 调用（每次一张），不进入扩散过程。
		// 工作流里没有任何图像输入节点，ComfyUI 侧也没有 IPAdapter /
		// ControlNet / CLIP-Vision 模型和自定义节点包。
		"referenceImageConditioning": map[string]any{
			"available": false,
			"influences": []string{
				"brief_understanding",
			},
			"reason": "workflow has no image input node; " +
				"no IPAdapter/ControlNet/CLIP-Vision model installed",
		},

		// 没有接 rembg / matting 一类的背景去除模型。合成器只按素材自带的
		// alpha 叠加，不会自己抠背景。
		"backgroundRemoval": map[string]any{
			"available": false,
			"reason": "no background removal model wired; " +
				"upload transparent PNG assets",
		},

		// 人物相似度需要人脸/图像嵌入模型，同样没有。
		"personSimilarityMetric": map[string]any{
			"available": false,
			"reason":    "no face/image embedding model installed",
		},
	}
}

func (s *Server) aiReady() bool {
	return s.aiClient != nil &&
		s.aiService != nil &&
		s.aiRuntime != nil
}
