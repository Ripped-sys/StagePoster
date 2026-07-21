package api

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Ripped-sys/StagePoster/backend/internal/repository"
	"github.com/Ripped-sys/StagePoster/backend/internal/service"
	"github.com/Ripped-sys/StagePoster/backend/internal/storage"
)

func (s *Server) handleAssets(
	writer http.ResponseWriter,
	request *http.Request,
) {
	switch request.Method {
	case http.MethodPost:
		s.handleAssetUpload(writer, request)

	case http.MethodGet:
		s.handleAssetList(writer, request)

	default:
		writeError(
			writer,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
	}
}

func (s *Server) handleAsset(
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
		"/api/assets/",
	)

	path = strings.Trim(path, "/")

	if path == "" {
		writeError(
			writer,
			http.StatusBadRequest,
			"asset id is required",
		)
		return
	}

	if strings.HasSuffix(path, "/content") {
		assetID := strings.TrimSuffix(path, "/content")
		assetID = strings.Trim(assetID, "/")

		s.handleAssetContent(
			writer,
			request,
			assetID,
		)
		return
	}

	if strings.Contains(path, "/") {
		writeError(
			writer,
			http.StatusNotFound,
			"asset route not found",
		)
		return
	}

	s.handleAssetMetadata(writer, request, path)
}

func (s *Server) handleAssetUpload(
	writer http.ResponseWriter,
	request *http.Request,
) {
	request.Body = http.MaxBytesReader(
		writer,
		request.Body,
		25<<20,
	)

	if err := request.ParseMultipartForm(8 << 20); err != nil {
		writeError(
			writer,
			http.StatusBadRequest,
			"invalid multipart upload: "+err.Error(),
		)
		return
	}

	if request.MultipartForm != nil {
		defer request.MultipartForm.RemoveAll()
	}

	file, header, err := request.FormFile("file")
	if err != nil {
		writeError(
			writer,
			http.StatusBadRequest,
			"file field is required",
		)
		return
	}
	defer file.Close()

	kind := request.FormValue("kind")

	ctx, cancel := contextWithTimeout(
		request,
		60*time.Second,
	)
	defer cancel()

	result, err := s.assetService.Upload(
		ctx,
		kind,
		header.Filename,
		file,
	)

	switch {
	case errors.Is(err, service.ErrInvalidAssetKind):
		writeError(
			writer,
			http.StatusBadRequest,
			"kind must be person, logo, or reference",
		)
		return

	case errors.Is(err, storage.ErrAssetTooLarge):
		writeError(
			writer,
			http.StatusRequestEntityTooLarge,
			"asset exceeds the 20 MB limit",
		)
		return

	case errors.Is(err, storage.ErrUnsupportedAssetType):
		writeError(
			writer,
			http.StatusUnsupportedMediaType,
			"only PNG and JPEG assets are supported",
		)
		return

	case errors.Is(err, storage.ErrEmptyAsset):
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

	writeJSON(writer, http.StatusCreated, result)
}

func (s *Server) handleAssetList(
	writer http.ResponseWriter,
	request *http.Request,
) {
	limit := 20

	if rawLimit := request.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
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

	result, err := s.assetService.List(ctx, limit)
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

func (s *Server) handleAssetMetadata(
	writer http.ResponseWriter,
	request *http.Request,
	assetID string,
) {
	ctx, cancel := contextWithTimeout(
		request,
		20*time.Second,
	)
	defer cancel()

	result, err := s.assetService.Get(ctx, assetID)

	if errors.Is(err, repository.ErrNotFound) {
		writeError(
			writer,
			http.StatusNotFound,
			"asset not found",
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

func (s *Server) handleAssetContent(
	writer http.ResponseWriter,
	request *http.Request,
	assetID string,
) {
	ctx, cancel := contextWithTimeout(
		request,
		60*time.Second,
	)
	defer cancel()

	result, err := s.assetService.OpenContent(ctx, assetID)

	if errors.Is(err, repository.ErrNotFound) {
		writeError(
			writer,
			http.StatusNotFound,
			"asset not found",
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
