package poster

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
)

var ErrUnsupportedStyle = errors.New("unsupported poster style")

type Planner struct{}

func NewPlanner() *Planner {
	return &Planner{}
}

func (p *Planner) Plan(
	request domain.CreatePosterRequest,
) ([]domain.CandidateSpec, error) {
	style := strings.TrimSpace(request.Visual.Style)

	if style == "" {
		style = "metal-gothic-v1"
	}

	if style != "metal-gothic-v1" {
		return nil, fmt.Errorf(
			"%w: %s",
			ErrUnsupportedStyle,
			style,
		)
	}

	return []domain.CandidateSpec{
		{
			VariantKey:  "monumental-empty-throne",
			VariantName: "Monumental Throne",
			Motif:       "an empty monumental winged throne assembled from blackened iron, broken amplifier cabinets and cathedral stone",
			Composition: "strict central symmetry with one dominant monumental emblem",
			Materials: []string{
				"black ink engraving",
				"distressed silkscreen",
				"oxidized iron",
				"cracked cathedral stone",
			},
			Palette: []string{
				"ink black",
				"dirty ivory",
				"oxide red",
			},
			Lighting: "severe frontal contrast with a restrained eclipse glow",
		},
		{
			VariantKey:  "mechanical-wolf-reliquary",
			VariantName: "Mechanical Totem",
			Motif:       "a mechanical wolf reliquary constructed from speaker cones, chains, metal ribs and ritual audio machinery",
			Composition: "triangular altar composition with the totem rising through the central field",
			Materials: []string{
				"brushed metal",
				"photocopied punk texture",
				"scratched aluminium",
				"rough screen print ink",
			},
			Palette: []string{
				"charcoal black",
				"silver grey",
				"bone white",
			},
			Lighting: "hard directional side light with sharp metallic highlights",
		},
		{
			VariantKey:  "cathedral-eclipse-portal",
			VariantName: "Cathedral Eclipse",
			Motif:       "a towering black cathedral portal framing a fractured eclipse and a ritual monument made from stacked loudspeakers",
			Composition: "vertical architectural composition with deep perspective and a central eclipse",
			Materials: []string{
				"weathered stone",
				"torn paper collage",
				"dry black ink",
				"distressed photocopy grain",
			},
			Palette: []string{
				"deep black",
				"smoke grey",
				"acid green accent",
			},
			Lighting: "backlit eclipse with deep architectural shadows",
		},
	}, nil
}

func (p *Planner) BuildPrompt(
	request domain.CreatePosterRequest,
	spec domain.CandidateSpec,
) string {
	theme := strings.TrimSpace(request.Visual.Theme)
	if theme == "" {
		theme = request.Event.Title
	}

	genre := strings.TrimSpace(request.Visual.MusicGenre)
	if genre == "" {
		genre = "underground heavy music"
	}

	mood := strings.Join(request.Visual.Mood, ", ")
	if mood == "" {
		mood = "dark, ritualistic, monumental"
	}

	preferredColors := strings.Join(
		request.Visual.PreferredColors,
		", ",
	)

	palette := strings.Join(spec.Palette, ", ")
	if preferredColors != "" {
		palette = preferredColors + ", informed by " + palette
	}

	materials := strings.Join(spec.Materials, ", ")

	return fmt.Sprintf(
		`A professional vertical 2:3 key visual for a %s music event themed "%s".

Create one dominant symbolic motif: %s. Use %s. The atmosphere is %s.

The composition must use a visually quiet title-safe area across the upper 18 percent, one powerful central key visual across the middle field, and a clean information-safe region across the lower 22 percent. The central visual must remain readable from a distance and richly detailed at close range.

Surface treatment and production language: %s. Color palette: %s. Lighting: %s.

The result must feel like original professional concert and festival artwork, not a movie screenshot, portrait, generic wallpaper, mockup, user interface or social-media meme.

No people, no performers, no faces, no human figures and no crowd. No readable typography, letters, numbers, captions, event names, brand names, logos, signatures or watermarks. The artwork is only the background key visual. Accurate titles, logos, dates, venue details and ticket information will be added later by a deterministic layout system.`,
		genre,
		theme,
		spec.Motif,
		spec.Composition,
		mood,
		materials,
		palette,
		spec.Lighting,
	)
}
