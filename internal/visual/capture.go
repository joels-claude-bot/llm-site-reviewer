// Package visual captures rendered pages for cognitive screenshot review.
package visual

import (
	"fmt"
	"math"
	"os"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
)

const (
	DefaultWidth     = 1280
	DefaultHeight    = 900
	DefaultStepRatio = 0.7
)

type Config struct {
	Width, Height int
	StepRatio     float64
	Settle        time.Duration
	Headless      bool
	Executable    string
}

type Frame struct {
	Index int
	Y     int
	PNG   []byte
}

type Capture struct {
	URL        string
	PageHeight int
	Width      int
	Height     int
	Frames     []Frame
}

func DefaultConfig() Config {
	return Config{Width: DefaultWidth, Height: DefaultHeight, StepRatio: DefaultStepRatio, Settle: 100 * time.Millisecond, Headless: true, Executable: os.Getenv("PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH")}
}

func ScrollPositions(pageHeight, viewportHeight int, stepRatio float64) ([]int, error) {
	if pageHeight <= 0 || viewportHeight <= 0 || stepRatio <= 0 || stepRatio > 1 {
		return nil, fmt.Errorf("invalid page segmentation dimensions or step ratio")
	}
	bottom := max(0, pageHeight-viewportHeight)
	step := max(1, int(math.Floor(float64(viewportHeight)*stepRatio)))
	positions := make([]int, 0, bottom/step+2)
	for y := 0; y < bottom; y += step {
		positions = append(positions, y)
	}
	minimumUsefulMove := max(1, viewportHeight-step)
	if len(positions) > 1 && bottom-positions[len(positions)-1] < minimumUsefulMove {
		positions = positions[:len(positions)-1]
	}
	if len(positions) == 0 || positions[len(positions)-1] != bottom {
		positions = append(positions, bottom)
	}
	return positions, nil
}

func CapturePage(url string, cfg Config) (_ Capture, err error) {
	pw, err := playwright.Run()
	if err != nil {
		return Capture{}, fmt.Errorf("start playwright: %w", err)
	}
	defer func() { _ = pw.Stop() }()
	launch := playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(cfg.Headless)}
	if cfg.Executable != "" {
		launch.ExecutablePath = playwright.String(cfg.Executable)
	}
	browser, err := pw.Chromium.Launch(launch)
	if err != nil {
		return Capture{}, fmt.Errorf("launch chromium: %w", err)
	}
	defer func() { _ = browser.Close() }()
	page, err := browser.NewPage(playwright.BrowserNewPageOptions{Viewport: &playwright.Size{Width: cfg.Width, Height: cfg.Height}})
	if err != nil {
		return Capture{}, fmt.Errorf("create page: %w", err)
	}
	if _, err = page.Goto(url, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); err != nil {
		return Capture{}, fmt.Errorf("navigate to %s: %w", url, err)
	}
	_, _ = page.Evaluate("() => document.fonts.ready")
	height, err := documentHeight(page)
	if err != nil {
		return Capture{}, err
	}
	positions, err := ScrollPositions(height, cfg.Height, cfg.StepRatio)
	if err != nil {
		return Capture{}, err
	}
	for _, y := range positions {
		_, _ = page.Evaluate("y => window.scrollTo(0, y)", y)
		page.WaitForTimeout(float64(cfg.Settle.Milliseconds()))
	}
	height, err = documentHeight(page)
	if err != nil {
		return Capture{}, err
	}
	positions, err = ScrollPositions(height, cfg.Height, cfg.StepRatio)
	if err != nil {
		return Capture{}, err
	}
	result := Capture{URL: url, PageHeight: height, Width: cfg.Width, Height: cfg.Height}
	for i, y := range positions {
		_, _ = page.Evaluate("y => window.scrollTo(0, y)", y)
		page.WaitForTimeout(float64(cfg.Settle.Milliseconds()))
		png, shotErr := page.Screenshot(playwright.PageScreenshotOptions{Animations: playwright.ScreenshotAnimationsDisabled, Caret: playwright.ScreenshotCaretHide})
		if shotErr != nil {
			return Capture{}, fmt.Errorf("capture frame %d: %w", i+1, shotErr)
		}
		result.Frames = append(result.Frames, Frame{Index: i + 1, Y: y, PNG: png})
	}
	return result, nil
}

func documentHeight(page playwright.Page) (int, error) {
	value, err := page.Evaluate("() => document.documentElement.scrollHeight")
	if err != nil {
		return 0, fmt.Errorf("read document height: %w", err)
	}
	height, ok := value.(int)
	if ok {
		return height, nil
	}
	if number, ok := value.(float64); ok {
		return int(number), nil
	}
	return 0, fmt.Errorf("unexpected document height type %T", value)
}
