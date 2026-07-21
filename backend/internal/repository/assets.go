package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
)

func (r *Repository) MigrateAssets(
	ctx context.Context,
) error {
	statements := []string{
		`
		CREATE TABLE IF NOT EXISTS assets (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,

			original_name TEXT NOT NULL,
			filename TEXT NOT NULL,
			mime_type TEXT NOT NULL,

			size_bytes INTEGER NOT NULL,
			sha256 TEXT NOT NULL,
			storage_path TEXT NOT NULL,

			width INTEGER NOT NULL DEFAULT 0,
			height INTEGER NOT NULL DEFAULT 0,

			created_at TEXT NOT NULL
		)
		`,
		`
		CREATE INDEX IF NOT EXISTS idx_assets_created_at
		ON assets(created_at DESC)
		`,
		`
		CREATE INDEX IF NOT EXISTS idx_assets_kind
		ON assets(kind, created_at DESC)
		`,
		`
		CREATE INDEX IF NOT EXISTS idx_assets_sha256
		ON assets(sha256)
		`,
	}

	for _, statement := range statements {
		if _, err := r.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf(
				"run assets migration: %w",
				err,
			)
		}
	}

	return nil
}

func (r *Repository) CreateAsset(
	ctx context.Context,
	asset domain.Asset,
) error {
	_, err := r.db.ExecContext(
		ctx,
		`
		INSERT INTO assets (
			id,
			kind,
			original_name,
			filename,
			mime_type,
			size_bytes,
			sha256,
			storage_path,
			width,
			height,
			created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
		asset.ID,
		asset.Kind,
		asset.OriginalName,
		asset.Filename,
		asset.MimeType,
		asset.SizeBytes,
		asset.SHA256,
		asset.StoragePath,
		asset.Width,
		asset.Height,
		formatTime(asset.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert asset: %w", err)
	}

	return nil
}

func (r *Repository) GetAsset(
	ctx context.Context,
	assetID string,
) (domain.Asset, error) {
	row := r.db.QueryRowContext(
		ctx,
		`
		SELECT
			id,
			kind,
			original_name,
			filename,
			mime_type,
			size_bytes,
			sha256,
			storage_path,
			width,
			height,
			created_at
		FROM assets
		WHERE id = ?
		`,
		assetID,
	)

	asset, err := scanAsset(row)

	if errors.Is(err, sql.ErrNoRows) {
		return domain.Asset{}, ErrNotFound
	}

	if err != nil {
		return domain.Asset{}, fmt.Errorf(
			"get asset: %w",
			err,
		)
	}

	return asset, nil
}

func (r *Repository) ListAssets(
	ctx context.Context,
	limit int,
) ([]domain.Asset, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			id,
			kind,
			original_name,
			filename,
			mime_type,
			size_bytes,
			sha256,
			storage_path,
			width,
			height,
			created_at
		FROM assets
		ORDER BY created_at DESC
		LIMIT ?
		`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list assets: %w",
			err,
		)
	}
	defer rows.Close()

	assets := make([]domain.Asset, 0, limit)

	for rows.Next() {
		asset, err := scanAsset(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"scan asset: %w",
				err,
			)
		}

		assets = append(assets, asset)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate assets: %w",
			err,
		)
	}

	return assets, nil
}

func scanAsset(source scanner) (domain.Asset, error) {
	var asset domain.Asset
	var createdAt string

	err := source.Scan(
		&asset.ID,
		&asset.Kind,
		&asset.OriginalName,
		&asset.Filename,
		&asset.MimeType,
		&asset.SizeBytes,
		&asset.SHA256,
		&asset.StoragePath,
		&asset.Width,
		&asset.Height,
		&createdAt,
	)
	if err != nil {
		return domain.Asset{}, err
	}

	asset.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return domain.Asset{}, err
	}

	return asset, nil
}
