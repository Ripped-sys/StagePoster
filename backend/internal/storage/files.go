package storage

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
)

type FileStore struct {
	root string
}

func NewFileStore(root string) (*FileStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("storage root is required")
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}

	return &FileStore{root: root}, nil
}

func (s *FileStore) SavePoster(
	jobID string,
	sourceFilename string,
	contentType string,
	body io.Reader,
) (domain.Output, error) {
	if !validID(jobID) {
		return domain.Output{}, fmt.Errorf("invalid job id")
	}

	mediaType := normalizeContentType(contentType)
	extension := outputExtension(sourceFilename, mediaType)

	jobDirectory := filepath.Join(s.root, jobID)
	if err := os.MkdirAll(jobDirectory, 0o755); err != nil {
		return domain.Output{}, fmt.Errorf(
			"create job storage directory: %w",
			err,
		)
	}

	filename := "poster" + extension
	finalPath := filepath.Join(jobDirectory, filename)

	tempFile, err := os.CreateTemp(jobDirectory, ".poster-*")
	if err != nil {
		return domain.Output{}, fmt.Errorf(
			"create temporary output file: %w",
			err,
		)
	}

	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := io.Copy(tempFile, body); err != nil {
		_ = tempFile.Close()
		return domain.Output{}, fmt.Errorf(
			"write poster output: %w",
			err,
		)
	}

	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return domain.Output{}, fmt.Errorf(
			"sync poster output: %w",
			err,
		)
	}

	if err := tempFile.Close(); err != nil {
		return domain.Output{}, fmt.Errorf(
			"close poster output: %w",
			err,
		)
	}

	if err := os.Rename(tempPath, finalPath); err != nil {
		return domain.Output{}, fmt.Errorf(
			"commit poster output: %w",
			err,
		)
	}

	width, height := imageDimensions(finalPath)

	outputID, err := domain.NewID("out_")
	if err != nil {
		return domain.Output{}, err
	}

	if mediaType == "" {
		mediaType = mime.TypeByExtension(extension)
	}

	if mediaType == "" {
		mediaType = "application/octet-stream"
	}

	return domain.Output{
		ID:          outputID,
		JobID:       jobID,
		Kind:        "poster",
		Filename:    filename,
		MimeType:    mediaType,
		StoragePath: finalPath,
		Width:       width,
		Height:      height,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

func (s *FileStore) Open(
	output domain.Output,
) (*os.File, error) {
	cleanRoot, err := filepath.Abs(s.root)
	if err != nil {
		return nil, err
	}

	cleanPath, err := filepath.Abs(output.StoragePath)
	if err != nil {
		return nil, err
	}

	relativePath, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return nil, err
	}

	if relativePath == ".." ||
		strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("output path is outside storage root")
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("open stored output: %w", err)
	}

	return file, nil
}

func normalizeContentType(value string) string {
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return strings.TrimSpace(value)
	}

	return mediaType
}

func outputExtension(
	sourceFilename string,
	mediaType string,
) string {
	sourceExtension := strings.ToLower(
		filepath.Ext(sourceFilename),
	)

	switch sourceExtension {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return sourceExtension
	}

	switch mediaType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}

func imageDimensions(path string) (int, int) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0
	}

	return config.Width, config.Height
}

func validID(value string) bool {
	if value == "" {
		return false
	}

	for _, character := range value {
		isLetter :=
			character >= 'a' && character <= 'z' ||
				character >= 'A' && character <= 'Z'

		isNumber :=
			character >= '0' && character <= '9'

		if !isLetter &&
			!isNumber &&
			character != '_' &&
			character != '-' {
			return false
		}
	}

	return true
}
