package composer

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	stdDraw "image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
	xDraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/webp"
)

var ErrInvalidCompositionInput = errors.New(
	"invalid poster composition input",
)

type Composer struct {
	outputRoot string
	regular    *opentype.Font
	bold       *opentype.Font
}

func New(
	outputRoot string,
	regularFontPath string,
	boldFontPath string,
) (*Composer, error) {
	outputRoot = strings.TrimSpace(outputRoot)

	if outputRoot == "" {
		return nil, errors.New(
			"poster output root is required",
		)
	}

	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return nil, fmt.Errorf(
			"create poster output root: %w",
			err,
		)
	}

	regular, err := loadFont(
		regularFontPath,
		goregular.TTF,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"load regular font: %w",
			err,
		)
	}

	bold, err := loadFont(
		boldFontPath,
		gobold.TTF,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"load bold font: %w",
			err,
		)
	}

	return &Composer{
		outputRoot: outputRoot,
		regular:    regular,
		bold:       bold,
	}, nil
}

func loadFont(
	path string,
	fallback []byte,
) (*opentype.Font, error) {
	data := fallback

	if strings.TrimSpace(path) != "" {
		loaded, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf(
				"read font %s: %w",
				path,
				err,
			)
		}

		data = loaded
	}

	parsed, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf(
			"parse font: %w",
			err,
		)
	}

	return parsed, nil
}

func (c *Composer) Compose(
	ctx context.Context,
	input domain.ComposeInput,
) (domain.ComposeResult, error) {
	if err := ctx.Err(); err != nil {
		return domain.ComposeResult{}, err
	}

	if err := validateComposeInput(input); err != nil {
		return domain.ComposeResult{}, err
	}

	if input.Width <= 0 {
		input.Width = 1024
	}

	if input.Height <= 0 {
		input.Height = 1536
	}

	keyVisual, err := decodeImage(input.KeyVisualPath)
	if err != nil {
		return domain.ComposeResult{},
			fmt.Errorf(
				"decode key visual: %w",
				err,
			)
	}

	canvas := image.NewNRGBA(
		image.Rect(
			0,
			0,
			input.Width,
			input.Height,
		),
	)

	xDraw.CatmullRom.Scale(
		canvas,
		canvas.Bounds(),
		keyVisual,
		keyVisual.Bounds(),
		xDraw.Src,
		nil,
	)

	if err := ctx.Err(); err != nil {
		return domain.ComposeResult{}, err
	}

	c.drawTopOverlay(canvas)

	adjustments := normalizeCompositionAdjustments(
		input.Adjustments,
	)

	panelTop := int(
		float64(input.Height) *
			adjustments.PanelTopRatio,
	)

	drawInformationBackground(
		canvas,
		panelTop,
		adjustments.PanelTheme,
	)

	if err := c.drawTitle(
		canvas,
		input.Event.Title,
		adjustments.TitleOffsetRatio,
	); err != nil {
		return domain.ComposeResult{}, err
	}

	if err := c.drawEventLogo(
		canvas,
		input.EventLogo,
	); err != nil {
		return domain.ComposeResult{}, err
	}

	if err := c.drawArtistIdentity(
		canvas,
		input.Event.Artist,
		input.ArtistLogo,
		panelTop,
	); err != nil {
		return domain.ComposeResult{}, err
	}

	if err := c.drawInformationPanel(
		canvas,
		input.Event,
		panelTop,
		adjustments.PanelTheme,
	); err != nil {
		return domain.ComposeResult{}, err
	}

	if err := c.drawSponsorLogos(
		canvas,
		input.Sponsors,
		panelTop,
	); err != nil {
		return domain.ComposeResult{}, err
	}

	if err := ctx.Err(); err != nil {
		return domain.ComposeResult{}, err
	}

	outputDirectory := filepath.Join(
		c.outputRoot,
		input.PosterID,
	)

	if err := os.MkdirAll(
		outputDirectory,
		0o755,
	); err != nil {
		return domain.ComposeResult{},
			fmt.Errorf(
				"create poster directory: %w",
				err,
			)
	}

	finalPath := filepath.Join(
		outputDirectory,
		"final-poster.png",
	)

	if err := writePNGAtomic(
		finalPath,
		canvas,
	); err != nil {
		return domain.ComposeResult{},
			fmt.Errorf(
				"write final poster: %w",
				err,
			)
	}

	thumbnailWidth := 512
	thumbnailHeight := input.Height *
		thumbnailWidth /
		input.Width

	thumbnail := image.NewNRGBA(
		image.Rect(
			0,
			0,
			thumbnailWidth,
			thumbnailHeight,
		),
	)

	xDraw.CatmullRom.Scale(
		thumbnail,
		thumbnail.Bounds(),
		canvas,
		canvas.Bounds(),
		xDraw.Src,
		nil,
	)

	thumbnailPath := filepath.Join(
		outputDirectory,
		"thumbnail.png",
	)

	if err := writePNGAtomic(
		thumbnailPath,
		thumbnail,
	); err != nil {
		return domain.ComposeResult{},
			fmt.Errorf(
				"write thumbnail: %w",
				err,
			)
	}

	return domain.ComposeResult{
		FinalPath:       finalPath,
		ThumbnailPath:   thumbnailPath,
		Width:           input.Width,
		Height:          input.Height,
		ThumbnailWidth:  thumbnailWidth,
		ThumbnailHeight: thumbnailHeight,
	}, nil
}

func normalizeCompositionAdjustments(
	adjustments domain.CompositionAdjustments,
) domain.CompositionAdjustments {
	switch strings.ToLower(
		strings.TrimSpace(adjustments.Template),
	) {
	case "editorial_top":
		if adjustments.TitleOffsetRatio == 0 {
			adjustments.TitleOffsetRatio = 0.035
		}

		if adjustments.PanelTopRatio == 0 {
			adjustments.PanelTopRatio = 0.80
		}

	case "cinematic_center":
		if adjustments.TitleOffsetRatio == 0 {
			adjustments.TitleOffsetRatio = 0.055
		}

		if adjustments.PanelTopRatio == 0 {
			adjustments.PanelTopRatio = 0.81
		}

		if strings.TrimSpace(
			adjustments.PanelTheme,
		) == "" {
			adjustments.PanelTheme = "dark"
		}

	case "gothic_frame":
		if adjustments.TitleOffsetRatio == 0 {
			adjustments.TitleOffsetRatio = 0.045
		}

		if adjustments.PanelTopRatio == 0 {
			adjustments.PanelTopRatio = 0.82
		}

		if strings.TrimSpace(
			adjustments.PanelTheme,
		) == "" {
			adjustments.PanelTheme = "dark"
		}
	}

	if adjustments.PanelTopRatio == 0 {
		adjustments.PanelTopRatio = 0.77
	}

	if adjustments.PanelTopRatio < 0.70 {
		adjustments.PanelTopRatio = 0.70
	}

	if adjustments.PanelTopRatio > 0.86 {
		adjustments.PanelTopRatio = 0.86
	}

	if adjustments.TitleOffsetRatio < 0 {
		adjustments.TitleOffsetRatio = 0
	}

	if adjustments.TitleOffsetRatio > 0.12 {
		adjustments.TitleOffsetRatio = 0.12
	}

	switch strings.ToLower(
		strings.TrimSpace(adjustments.PanelTheme),
	) {
	case "dark":
		adjustments.PanelTheme = "dark"

	default:
		adjustments.PanelTheme = "light"
	}

	return adjustments
}

func informationPanelColors(
	theme string,
) (
	panel color.NRGBA,
	accent color.NRGBA,
	value color.NRGBA,
	label color.NRGBA,
) {
	accent = color.NRGBA{
		R: 139,
		G: 23,
		B: 30,
		A: 255,
	}

	if strings.EqualFold(
		strings.TrimSpace(theme),
		"dark",
	) {
		return color.NRGBA{
				R: 14,
				G: 14,
				B: 15,
				A: 255,
			},
			accent,
			color.NRGBA{
				R: 242,
				G: 237,
				B: 222,
				A: 255,
			},
			color.NRGBA{
				R: 211,
				G: 74,
				B: 80,
				A: 255,
			}
	}

	return color.NRGBA{
			R: 236,
			G: 231,
			B: 216,
			A: 255,
		},
		accent,
		color.NRGBA{
			R: 19,
			G: 19,
			B: 18,
			A: 255,
		},
		accent
}

func validateComposeInput(
	input domain.ComposeInput,
) error {
	if strings.TrimSpace(input.PosterID) == "" {
		return fmt.Errorf(
			"%w: poster id is required",
			ErrInvalidCompositionInput,
		)
	}

	if filepath.Base(input.PosterID) !=
		input.PosterID {
		return fmt.Errorf(
			"%w: poster id contains path characters",
			ErrInvalidCompositionInput,
		)
	}

	if strings.TrimSpace(input.CandidateID) == "" {
		return fmt.Errorf(
			"%w: candidate id is required",
			ErrInvalidCompositionInput,
		)
	}

	if strings.TrimSpace(input.KeyVisualPath) == "" {
		return fmt.Errorf(
			"%w: key visual path is required",
			ErrInvalidCompositionInput,
		)
	}

	if strings.TrimSpace(input.Event.Title) == "" {
		return fmt.Errorf(
			"%w: event title is required",
			ErrInvalidCompositionInput,
		)
	}

	return nil
}

func drawInformationBackground(
	canvas *image.NRGBA,
	panelTop int,
	theme string,
) {
	panelColor, accentColor, _, _ :=
		informationPanelColors(theme)

	stdDraw.Draw(
		canvas,
		image.Rect(
			0,
			panelTop,
			canvas.Bounds().Dx(),
			canvas.Bounds().Dy(),
		),
		image.NewUniform(panelColor),
		image.Point{},
		stdDraw.Src,
	)

	stdDraw.Draw(
		canvas,
		image.Rect(
			0,
			panelTop,
			canvas.Bounds().Dx(),
			panelTop+10,
		),
		image.NewUniform(accentColor),
		image.Point{},
		stdDraw.Src,
	)
}

func (c *Composer) drawTopOverlay(
	canvas *image.NRGBA,
) {
	height := int(
		float64(canvas.Bounds().Dy()) * 0.24,
	)

	if height <= 0 {
		return
	}

	for y := 0; y < height; y++ {
		ratio := 1 -
			float64(y)/
				float64(height)

		alpha := uint8(190 * ratio)

		stdDraw.Draw(
			canvas,
			image.Rect(
				0,
				y,
				canvas.Bounds().Dx(),
				y+1,
			),
			image.NewUniform(
				color.NRGBA{
					R: 0,
					G: 0,
					B: 0,
					A: alpha,
				},
			),
			image.Point{},
			stdDraw.Over,
		)
	}
}

func (c *Composer) drawTitle(
	canvas *image.NRGBA,
	title string,
	offsetRatio float64,
) error {
	title = strings.TrimSpace(title)

	if title == "" {
		return nil
	}

	maxWidth := int(
		float64(canvas.Bounds().Dx()) * 0.82,
	)

	maxHeight := int(
		float64(canvas.Bounds().Dy()) * 0.15,
	)

	face, lines, err := c.fitWrappedFace(
		c.bold,
		title,
		maxWidth,
		maxHeight,
		2,
		88,
		38,
	)
	if err != nil {
		return fmt.Errorf(
			"create title font: %w",
			err,
		)
	}
	defer closeFace(face)

	lineHeight := face.Metrics().
		Height.
		Ceil()

	totalHeight := lineHeight * len(lines)

	startY := 48 +
		int(
			float64(canvas.Bounds().Dy())*
				offsetRatio,
		) +
		(maxHeight-totalHeight)/2 +
		face.Metrics().
			Ascent.
			Ceil()

	for index, line := range lines {
		drawCenteredText(
			canvas,
			face,
			line,
			canvas.Bounds().Dx()/2,
			startY+index*lineHeight,
			color.White,
		)
	}

	return nil
}

func (c *Composer) drawEventLogo(
	canvas *image.NRGBA,
	logo domain.CompositionAsset,
) error {
	if logo.StoragePath == "" {
		return nil
	}

	target := image.Rect(
		int(float64(canvas.Bounds().Dx())*0.76),
		int(float64(canvas.Bounds().Dy())*0.035),
		int(float64(canvas.Bounds().Dx())*0.95),
		int(float64(canvas.Bounds().Dy())*0.12),
	)

	if err := c.drawAssetContained(
		canvas,
		logo,
		target,
	); err != nil {
		return fmt.Errorf(
			"compose event logo: %w",
			err,
		)
	}

	return nil
}

func (c *Composer) drawArtistIdentity(
	canvas *image.NRGBA,
	artist string,
	logo domain.CompositionAsset,
	panelTop int,
) error {
	if logo.StoragePath != "" {
		target := image.Rect(
			int(float64(canvas.Bounds().Dx())*0.19),
			int(float64(canvas.Bounds().Dy())*0.63),
			int(float64(canvas.Bounds().Dx())*0.81),
			int(float64(canvas.Bounds().Dy())*0.745),
		)

		if err := c.drawAssetContained(
			canvas,
			logo,
			target,
		); err != nil {
			return fmt.Errorf(
				"compose artist logo: %w",
				err,
			)
		}

		return nil
	}

	artist = strings.TrimSpace(artist)

	if artist == "" {
		return nil
	}

	face, err := c.fitSingleLineFace(
		c.bold,
		artist,
		int(float64(canvas.Bounds().Dx())*0.72),
		66,
		28,
	)
	if err != nil {
		return fmt.Errorf(
			"create artist font: %w",
			err,
		)
	}
	defer closeFace(face)

	drawCenteredText(
		canvas,
		face,
		artist,
		canvas.Bounds().Dx()/2,
		panelTop-52,
		color.White,
	)

	return nil
}

func (c *Composer) drawInformationPanel(
	canvas *image.NRGBA,
	event domain.EventBrief,
	panelTop int,
	theme string,
) error {
	labelFace, err := newFace(
		c.regular,
		17,
	)
	if err != nil {
		return err
	}
	defer closeFace(labelFace)

	valueFace, err := newFace(
		c.bold,
		31,
	)
	if err != nil {
		return err
	}
	defer closeFace(valueFace)

	_, _, dark, red :=
		informationPanelColors(theme)

	leftX := 66
	centerX := canvas.Bounds().Dx() / 2
	rightX := canvas.Bounds().Dx() - 66

	labelY := panelTop + 64
	valueY := panelTop + 108

	drawText(
		canvas,
		labelFace,
		"DATE / TIME",
		leftX,
		labelY,
		red,
	)

	dateTime := joinNonEmpty(
		strings.TrimSpace(event.Date),
		strings.TrimSpace(event.Time),
	)

	drawText(
		canvas,
		valueFace,
		dateTime,
		leftX,
		valueY,
		dark,
	)

	drawCenteredText(
		canvas,
		labelFace,
		"VENUE",
		centerX,
		labelY,
		red,
	)

	venueFace, err := c.fitSingleLineFace(
		c.bold,
		strings.TrimSpace(event.Venue),
		int(float64(canvas.Bounds().Dx())*0.34),
		31,
		18,
	)
	if err != nil {
		return err
	}
	defer closeFace(venueFace)

	drawCenteredText(
		canvas,
		venueFace,
		strings.TrimSpace(event.Venue),
		centerX,
		valueY,
		dark,
	)

	drawRightText(
		canvas,
		labelFace,
		"TICKETS",
		rightX,
		labelY,
		red,
	)

	price := ticketText(event)

	priceFace, err := c.fitSingleLineFace(
		c.bold,
		price,
		int(float64(canvas.Bounds().Dx())*0.31),
		29,
		16,
	)
	if err != nil {
		return err
	}
	defer closeFace(priceFace)

	drawRightText(
		canvas,
		priceFace,
		price,
		rightX,
		valueY,
		dark,
	)

	return nil
}

func joinNonEmpty(
	values ...string,
) string {
	result := make(
		[]string,
		0,
		len(values),
	)

	for _, value := range values {
		value = strings.TrimSpace(value)

		if value != "" {
			result = append(
				result,
				value,
			)
		}
	}

	return strings.Join(result, "  ")
}

func ticketText(
	event domain.EventBrief,
) string {
	parts := make([]string, 0, 2)

	if strings.TrimSpace(
		event.PresalePrice,
	) != "" {
		parts = append(
			parts,
			"PRE "+strings.TrimSpace(
				event.PresalePrice,
			),
		)
	}

	if strings.TrimSpace(
		event.DoorPrice,
	) != "" {
		parts = append(
			parts,
			"DOOR "+strings.TrimSpace(
				event.DoorPrice,
			),
		)
	}

	if len(parts) == 0 {
		return "TICKETS AT VENUE"
	}

	return strings.Join(parts, " / ")
}

func (c *Composer) drawSponsorLogos(
	canvas *image.NRGBA,
	sponsors []domain.CompositionAsset,
	panelTop int,
) error {
	if len(sponsors) == 0 {
		return nil
	}

	if len(sponsors) > 5 {
		sponsors = sponsors[:5]
	}

	totalWidth := int(
		float64(canvas.Bounds().Dx()) * 0.76,
	)

	gap := 18

	itemWidth := (totalWidth - gap*(len(sponsors)-1)) / len(sponsors)

	startX := (canvas.Bounds().Dx() - totalWidth) / 2

	top := panelTop + 190
	bottom := canvas.Bounds().Dy() - 28

	if bottom <= top {
		return nil
	}

	for index, sponsor := range sponsors {
		left := startX +
			index*(itemWidth+gap)

		if err := c.drawAssetContained(
			canvas,
			sponsor,
			image.Rect(
				left,
				top,
				left+itemWidth,
				bottom,
			),
		); err != nil {
			return fmt.Errorf(
				"compose sponsor logo %s: %w",
				sponsor.ID,
				err,
			)
		}
	}

	return nil
}

func (c *Composer) drawAssetContained(
	canvas *image.NRGBA,
	asset domain.CompositionAsset,
	target image.Rectangle,
) error {
	if strings.TrimSpace(
		asset.StoragePath,
	) == "" {
		return nil
	}

	if target.Empty() {
		return errors.New(
			"logo target rectangle is empty",
		)
	}

	if strings.Contains(
		strings.ToLower(asset.MimeType),
		"svg",
	) {
		return errors.New(
			"SVG composition is not supported by Composer v1; upload a transparent PNG",
		)
	}

	source, err := decodeImage(
		asset.StoragePath,
	)
	if err != nil {
		return err
	}

	sourceWidth := source.Bounds().Dx()
	sourceHeight := source.Bounds().Dy()

	if sourceWidth <= 0 ||
		sourceHeight <= 0 {
		return errors.New(
			"logo has invalid dimensions",
		)
	}

	scaleX := float64(target.Dx()) /
		float64(sourceWidth)

	scaleY := float64(target.Dy()) /
		float64(sourceHeight)

	scale := scaleX

	if scaleY < scale {
		scale = scaleY
	}

	width := int(
		float64(sourceWidth) * scale,
	)

	height := int(
		float64(sourceHeight) * scale,
	)

	if width <= 0 || height <= 0 {
		return errors.New(
			"scaled logo has invalid dimensions",
		)
	}

	left := target.Min.X +
		(target.Dx()-width)/2

	top := target.Min.Y +
		(target.Dy()-height)/2

	xDraw.CatmullRom.Scale(
		canvas,
		image.Rect(
			left,
			top,
			left+width,
			top+height,
		),
		source,
		source.Bounds(),
		xDraw.Over,
		nil,
	)

	return nil
}

func decodeImage(
	path string,
) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoded, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

func writePNGAtomic(
	path string,
	value image.Image,
) error {
	temporary := path + ".tmp"

	file, err := os.Create(temporary)
	if err != nil {
		return err
	}

	encodeErr := png.Encode(file, value)
	closeErr := file.Close()

	if encodeErr != nil {
		_ = os.Remove(temporary)
		return encodeErr
	}

	if closeErr != nil {
		_ = os.Remove(temporary)
		return closeErr
	}

	if err := os.Rename(
		temporary,
		path,
	); err != nil {
		_ = os.Remove(temporary)
		return err
	}

	return nil
}

func newFace(
	value *opentype.Font,
	size float64,
) (font.Face, error) {
	if value == nil {
		return nil, errors.New(
			"font is nil",
		)
	}

	return opentype.NewFace(
		value,
		&opentype.FaceOptions{
			Size:    size,
			DPI:     72,
			Hinting: font.HintingFull,
		},
	)
}

func closeFace(
	face font.Face,
) {
	if closer, ok := face.(interface {
		Close() error
	}); ok {
		_ = closer.Close()
	}
}

func (c *Composer) fitSingleLineFace(
	value *opentype.Font,
	text string,
	maxWidth int,
	startSize float64,
	minSize float64,
) (font.Face, error) {
	text = strings.TrimSpace(text)

	for size := startSize; size >= minSize; size -= 2 {
		face, err := newFace(
			value,
			size,
		)
		if err != nil {
			return nil, err
		}

		if font.MeasureString(
			face,
			text,
		).Ceil() <= maxWidth {
			return face, nil
		}

		closeFace(face)
	}

	return newFace(value, minSize)
}

func (c *Composer) fitWrappedFace(
	value *opentype.Font,
	text string,
	maxWidth int,
	maxHeight int,
	maxLines int,
	startSize float64,
	minSize float64,
) (font.Face, []string, error) {
	for size := startSize; size >= minSize; size -= 2 {
		face, err := newFace(
			value,
			size,
		)
		if err != nil {
			return nil, nil, err
		}

		lines := wrapText(
			face,
			text,
			maxWidth,
		)

		height := len(lines) *
			face.Metrics().
				Height.
				Ceil()

		if len(lines) <= maxLines &&
			height <= maxHeight {
			return face, lines, nil
		}

		closeFace(face)
	}

	face, err := newFace(
		value,
		minSize,
	)
	if err != nil {
		return nil, nil, err
	}

	lines := wrapText(
		face,
		text,
		maxWidth,
	)

	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	return face, lines, nil
}

func wrapText(
	face font.Face,
	text string,
	maxWidth int,
) []string {
	text = strings.TrimSpace(text)

	if text == "" {
		return nil
	}

	words := strings.Fields(text)

	if len(words) <= 1 {
		return wrapRunes(
			face,
			text,
			maxWidth,
		)
	}

	lines := make([]string, 0, 2)
	current := words[0]

	for _, word := range words[1:] {
		candidate := current +
			" " +
			word

		if font.MeasureString(
			face,
			candidate,
		).Ceil() <= maxWidth {
			current = candidate
			continue
		}

		lines = append(
			lines,
			current,
		)

		current = word
	}

	lines = append(lines, current)

	return lines
}

func wrapRunes(
	face font.Face,
	text string,
	maxWidth int,
) []string {
	runes := []rune(text)

	if len(runes) == 0 {
		return nil
	}

	lines := make([]string, 0, 2)
	current := ""

	for _, value := range runes {
		candidate := current +
			string(value)

		if current != "" &&
			font.MeasureString(
				face,
				candidate,
			).Ceil() > maxWidth {
			lines = append(
				lines,
				current,
			)

			current = string(value)
			continue
		}

		current = candidate
	}

	if current != "" {
		lines = append(
			lines,
			current,
		)
	}

	return lines
}

func drawText(
	target stdDraw.Image,
	face font.Face,
	text string,
	x int,
	baseline int,
	value color.Color,
) {
	if face == nil ||
		strings.TrimSpace(text) == "" {
		return
	}

	drawer := &font.Drawer{
		Dst:  target,
		Src:  image.NewUniform(value),
		Face: face,
		Dot: fixed.Point26_6{
			X: fixed.I(x),
			Y: fixed.I(baseline),
		},
	}

	drawer.DrawString(text)
}

func drawCenteredText(
	target stdDraw.Image,
	face font.Face,
	text string,
	centerX int,
	baseline int,
	value color.Color,
) {
	width := font.MeasureString(
		face,
		text,
	).Ceil()

	drawText(
		target,
		face,
		text,
		centerX-width/2,
		baseline,
		value,
	)
}

func drawRightText(
	target stdDraw.Image,
	face font.Face,
	text string,
	rightX int,
	baseline int,
	value color.Color,
) {
	width := font.MeasureString(
		face,
		text,
	).Ceil()

	drawText(
		target,
		face,
		text,
		rightX-width,
		baseline,
		value,
	)
}
