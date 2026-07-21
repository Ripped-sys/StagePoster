package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

  "github.com/Ripped-sys/StagePoster/backend/internal/api"
  "github.com/Ripped-sys/StagePoster/backend/internal/comfy"
  "github.com/Ripped-sys/StagePoster/backend/internal/service"
)

func main() {
	listenAddr := env("LISTEN_ADDR", ":8080")
	comfyURL := env(
		"COMFY_URL",
		"http://127.0.0.1:8188",
	)
	workflowPath := env(
		"WORKFLOW_PATH",
		"/workspace/poster-engine/workflows/z_image_poster_v1.json",
	)

	template, err := comfy.LoadTemplate(
		workflowPath,
		os.Getenv("PROMPT_NODE_ID"),
		os.Getenv("NEGATIVE_PROMPT_NODE_ID"),
		os.Getenv("SEED_NODE_ID"),
	)
	if err != nil {
		log.Fatalf("load workflow template: %v", err)
	}

	comfyClient := comfy.NewClient(comfyURL)

	posterService := service.NewPosterService(
		comfyClient,
		template,
	)

	apiServer := api.NewServer(
		posterService,
		os.Getenv("POSTER_API_TOKEN"),
		env("CORS_ORIGIN", "*"),
	)

	log.Printf("Poster backend listening on %s", listenAddr)
	log.Printf("ComfyUI URL: %s", comfyURL)
	log.Printf("Workflow: %s", workflowPath)
	log.Printf("Bindings: %+v", template.Bindings())

	server := &http.Server{
		Addr:              listenAddr,
		Handler:           apiServer.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       90 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}
