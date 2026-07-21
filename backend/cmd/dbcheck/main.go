package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Ripped-sys/StagePoster/backend/internal/repository"
)

func main() {
	databasePath := os.Getenv("DB_PATH")
	if databasePath == "" {
		databasePath = "/workspace/poster-engine/backend/data/poster.db"
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	repositoryInstance, err := repository.OpenSQLite(
		ctx,
		databasePath,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer repositoryInstance.Close()

	jobs, err := repositoryInstance.ListJobs(ctx, 5)
	if err != nil {
		log.Fatal(err)
	}

	result := map[string]any{
		"status":       "ok",
		"databasePath": databasePath,
		"recentJobs":   jobs,
		"jobCount":     len(jobs),
	}

	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(raw))
}
