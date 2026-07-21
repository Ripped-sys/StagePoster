package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Ripped-sys/StagePoster/backend/internal/ai"
	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
)

func main() {
	client := ai.NewClient(
		env(
			"VLM_URL",
			"http://127.0.0.1:8001",
		),
		env(
			"VLM_API_KEY",
			"stageposter-vlm-local",
		),
		env(
			"VLM_MODEL",
			"stageposter-vlm",
		),
		120*time.Second,
	)

	service := ai.NewService(client)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Minute,
	)
	defer cancel()

	if err := service.Health(ctx); err != nil {
		log.Fatalf("VLM health failed: %v", err)
	}

	event := domain.EventBrief{
		Title:        "Abyssal Kingdom Festival",
		Artist:       "Maverick",
		Date:         "2026-08-21",
		Time:         "20:00",
		Venue:        "Void Arena",
		PresalePrice: "$45",
		DoorPrice:    "$60",
	}

	visual := domain.VisualBrief{
		Style:      "dark fantasy editorial",
		Theme:      "abyssal gothic kingdom",
		MusicGenre: "gothic metal",
		Mood: []string{
			"epic",
			"mysterious",
			"ritualistic",
		},
		PreferredColors: []string{
			"black",
			"aged ivory",
			"deep red",
		},
	}

	plans, metrics, err := service.Plan(
		ctx,
		event,
		visual,
		"黑色王座、巨大羽翼、哥特王国，整体要高级且有压迫感。",
	)
	if err != nil {
		log.Fatalf("design plan failed: %v", err)
	}

	fmt.Printf(
		"DESIGN latency=%s tokens=%d\n",
		metrics.Latency,
		metrics.TotalTokens,
	)

	printJSON(plans)

	reviewImage := os.Getenv("REVIEW_IMAGE")
	if reviewImage == "" {
		return
	}

	review, reviewMetrics, err := service.Review(
		ctx,
		reviewImage,
		event,
		visual,
		&plans.Plans[0],
	)
	if err != nil {
		log.Fatalf("review failed: %v", err)
	}

	fmt.Printf(
		"REVIEW latency=%s tokens=%d\n",
		reviewMetrics.Latency,
		reviewMetrics.TotalTokens,
	)

	printJSON(review)
}

func printJSON(
	value any,
) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(value); err != nil {
		log.Fatal(err)
	}
}

func env(
	key string,
	fallback string,
) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
