package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/gorilla/websocket"
)

const (
	canvasWidth  = 64 // Canvas dimensions
	canvasHeight = 32
	cellSize     = 3 // Each cell is 3x3 pixels
)

// FrameSnake defines the structure of a snake in a game frame
type FrameSnake struct {
	ID            string  `json:"ID"`
	Name          string  `json:"Name"`
	Body          []Point `json:"Body"`
	Health        int     `json:"Health"`
	Color         string  `json:"Color"`
	HeadType      string  `json:"HeadType"`
	TailType      string  `json:"TailType"`
	Latency       string  `json:"Latency"`
	Shout         string  `json:"Shout"`
	IsBot         bool    `json:"IsBot"`
	IsEnvironment bool    `json:"IsEnvironment"`
	Author        string  `json:"Author"`
	Death         *Death  `json:"Death"` // Add Death field (can be nil if not dead)
}

// Death defines the structure of a death event in a snake's life
type Death struct {
	Cause        string `json:"Cause"`
	Turn         int    `json:"Turn"`
	EliminatedBy string `json:"EliminatedBy"`
}

// FrameEvent defines the event structure including the list of snakes
type FrameEvent struct {
	Type string `json:"Type"`
	Data struct {
		ID     string       `json:"ID"`
		Turn   int          `json:"Turn"`
		Snakes []FrameSnake `json:"Snakes"` // FrameSnake for event snakes
		Food   []Point      `json:"Food"`
		Width  int          `json:"Width"`  // Board width
		Height int          `json:"Height"` // Board height
	} `json:"Data"`
}

func RetrieveGameRenderAndSendToTidbyt(gameID string) {

	// WebSocket URL for the game
	wsURL := fmt.Sprintf("wss://engine.battlesnake.com/games/%s/events", gameID)

	// Collect game frames
	frames, won, err := collectGameFrames(wsURL)
	if err != nil {
		slog.Error("Failed to collect game frames", "error", err.Error())
	}
	slog.Info("got frames from websocket", "turns", len(frames))

	// Render frames to WebP and push to Tidbyt
	err = renderGameToGIF(frames, deviceID, won)
	if err != nil {
		slog.Error("Failed to render game to gif", "error", err.Error())
	}

}

// Generate color from a hash of the snake name
func generateColor(name string) color.RGBA {
	h := sha1.New()
	h.Write([]byte(name))
	hash := h.Sum(nil)
	return color.RGBA{hash[0], hash[1], hash[2], 255}
}

// Lighten a color (used for snake heads)
func lighten(c color.RGBA) color.RGBA {
	return color.RGBA{
		R: uint8(min(int(c.R)+30, 255)),
		G: uint8(min(int(c.G)+30, 255)),
		B: uint8(min(int(c.B)+30, 255)),
		A: c.A,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Collect game frames from WebSocket and save board dimensions from the `game_end` event
func collectGameFrames(wsURL string) ([]*Board, bool, error) {
	var boards []*Board
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()
	var boardWidth, boardHeight int
	var gregoryWon bool

	var lastFrameEvent FrameEvent
	for {
		_, message, err := conn.ReadMessage()
		if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
			break
		} else if err != nil {
			return nil, false, fmt.Errorf("error reading message: %v", err)
		}

		var event FrameEvent

		if err := json.Unmarshal(message, &event); err != nil {
			slog.Error("Failed to unmarshal frame", "error", err.Error())
			continue
		}

		// Check for game_end event
		if event.Type == "game_end" {
			boardWidth = event.Data.Width
			boardHeight = event.Data.Height
			// Check if Gregory won (i.e., no Death value)
			break
		}
		lastFrameEvent = event

		board := &Board{
			Snakes: convertFrameEventToGame(event),
			Food:   event.Data.Food,
		}
		boards = append(boards, board)

	}

	for _, snake := range lastFrameEvent.Data.Snakes {
		if snake.Name == "Gregory" && snake.Death == nil {
			gregoryWon = true
			break
		}
	}

	// update the game dimensions in every frame
	for _, board := range boards {
		board.Height = boardHeight
		board.Width = boardWidth
	}

	return boards, gregoryWon, nil
}

// Conversion function: FrameSnake -> GameSnake
func convertFrameSnakeToGameSnake(fs FrameSnake) Snake {
	// The head is the first element of the body array in the frame snake object
	head := Point{}
	if len(fs.Body) > 0 {
		head = fs.Body[0]
	}

	// Convert FrameSnake to GameSnake
	return Snake{
		ID:      fs.ID,
		Name:    fs.Name,
		Health:  fs.Health,
		Body:    fs.Body,
		Latency: fs.Latency,
		Head:    head, // Head is the first element of the body
		Shout:   fs.Shout,
		Customizations: Customizations{
			Color: fs.Color,
			Head:  fs.HeadType,
			Tail:  fs.TailType,
		},
	}
}

// Example usage within collectGameFrames or anywhere else
func convertFrameEventToGame(frameEvent FrameEvent) []Snake {
	var gameSnakes []Snake
	for _, frameSnake := range frameEvent.Data.Snakes {
		gameSnake := convertFrameSnakeToGameSnake(frameSnake)
		gameSnakes = append(gameSnakes, gameSnake)
	}
	return gameSnakes
}

// Render a single board to an image with 3x3 pixel cells, border, y-axis flip, and snake names
func renderBoardToImage(board *Board) (*image.RGBA, []color.Color) {
	palette := []color.Color{
		color.RGBA{0, 0, 0, 255},       // Black
		color.RGBA{255, 255, 255, 255}, // White
		color.RGBA{255, 0, 0, 255},     // Red
		color.RGBA{0, 255, 0, 255},     // Green
		color.RGBA{0, 0, 255, 255},     // Blue
		color.RGBA{100, 100, 100, 255}, // Grey
	}

	img := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))

	// Fill the background with black
	black := color.RGBA{0, 0, 0, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{black}, image.Point{}, draw.Src)

	// Calculate the offset to move the board to the far right
	offsetX := canvasWidth - board.Width*3 // The far-right position, considering 3x3 cells
	offsetY := 0
	dividerColor := color.RGBA{100, 100, 100, 255}
	dividerRect := image.Rect(canvasWidth-3*board.Width-1, 0, canvasWidth-3*board.Width, canvasHeight)
	draw.Draw(img, dividerRect, &image.Uniform{dividerColor}, image.Point{}, draw.Src)

	// Draw the snakes
	// Render snake names on the left side
	yOffset := 10
	for _, snake := range board.Snakes {
		bodyColor, err := hexToRGBA(snake.Customizations.Color)
		if err != nil {
			bodyColor = generateColor(snake.Name)
		}
		headColor := lighten(bodyColor)
		palette = append(palette, bodyColor)
		palette = append(palette, headColor)

		// Draw snake's body
		for i, segment := range snake.Body {
			flippedY := board.Height - 1 - segment.Y // Flip along Y axis

			if i == 0 {
				// Head of the snake (slightly lighter)
				drawCell(img, offsetX+segment.X*3, offsetY+flippedY*3, headColor)
			} else {
				// Body of the snake
				drawCell(img, offsetX+segment.X*3, offsetY+flippedY*3, bodyColor)
			}
		}

		addScaledLabel(img, 10, yOffset, fmt.Sprintf("%3d", len(snake.Body)), bodyColor) // Render each snake name starting from (10, yOffset)
		yOffset += 20
	}

	// Draw food (in green)
	green := color.RGBA{0, 255, 0, 255}
	for _, food := range board.Food {
		flippedY := board.Height - 1 - food.Y // Flip along Y axis
		drawCell(img, offsetX+food.X*3, offsetY+flippedY*3, green)
	}

	return img, palette
}

// Helper function to add text (snake names) using the basic font
func addScaledLabel(img *image.RGBA, x, y int, label string, col color.RGBA) {
	point := fixed.Point26_6{
		X: fixed.I(x),
		Y: fixed.I(y),
	}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}

// Convert hex string (e.g., "#FF5733" or "FF5733") to color.RGBA
func hexToRGBA(hex string) (color.RGBA, error) {
	// Remove the '#' if it's present
	hex = strings.TrimPrefix(hex, "#")

	// Parse the hex string, which should be 6 characters long (RRGGBB)
	if len(hex) != 6 {
		return color.RGBA{}, fmt.Errorf("invalid hex color format: %s", hex)
	}

	// Parse the individual components from the hex string
	r, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}
	g, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}
	b, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return color.RGBA{}, err
	}

	// Return the color.RGBA object (fully opaque, so A = 255)
	return color.RGBA{uint8(r), uint8(g), uint8(b), 255}, nil
}

// Draw a 3x3 cell at the specified board position, accounting for centering
func drawCell(img *image.RGBA, x, y int, c color.RGBA) {
	// Each "cell" is now 3x3 pixels, so expand each cell to fill that space
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if y+j < canvasHeight { // Ensure we don't draw outside the canvas height
				img.Set(x+i, y+j, c)
			}
		}
	}
}

// Stitch together frames and encode as GIF animation with dynamic delay to fit within 15 seconds
func renderGameToGIF(frames []*Board, deviceID string, gregoryWon bool) error {

	if len(frames) == 0 {
		slog.Warn("no frames to be rendered")
		return nil
	}

	slog.Info("rendering game")
	totalDuration := 13000                               // 15 seconds in milliseconds
	maxDelayPerFrame := 20                               // Maximum delay of 200ms (200ms = 20 * 10ms)
	framesPerChunk := len(frames)                        // Total number of frames in the game
	delayPerFrame := totalDuration / framesPerChunk / 10 // Calculate the delay dynamically

	// Cap the delay to ensure it's not longer than 200ms per frame
	if delayPerFrame > maxDelayPerFrame {
		delayPerFrame = maxDelayPerFrame
	}

	// Arrays to store the full set of images and delays for the entire GIF
	var images []*image.Paletted
	var delays []int

	// Loop through each board (frame) and render it
	for i, board := range frames {
		img, palette := renderBoardToImage(board)

		// Convert the image to a paletted image (required for GIFs)
		palettedImage := image.NewPaletted(img.Bounds(), palette)
		draw.FloydSteinberg.Draw(palettedImage, img.Bounds(), img, image.Point{})

		// Append the paletted image and the dynamic delay (in 100ths of a second)
		images = append(images, palettedImage)
		if i == len(frames)-1 {
			delays = append(delays, 200) // longer delay on last frame
		} else {
			delays = append(delays, delayPerFrame) // Dynamic delay per frame
		}
	}

	// If Gregory won, append a green screen at the end, otherwise append a red screen
	var winScreenPalette color.Palette
	if gregoryWon {
		winScreenPalette = color.Palette{color.RGBA{0, 255, 0, 255}}
	} else {
		winScreenPalette = color.Palette{color.RGBA{255, 0, 0, 255}}
	}
	// Create the win/lose screen as a paletted image
	finalScreen := image.NewPaletted(image.Rect(0, 0, canvasWidth, canvasHeight), winScreenPalette)

	// Append the final screen image with a delay of 1 second (100 * 10ms = 1000ms)
	images = append(images, finalScreen)
	delays = append(delays, 100) // 1 second delay for the final screen

	// Create a buffer to store the full GIF data
	var buf bytes.Buffer

	// Encode the images (including the final screen) into a single GIF
	err := gif.EncodeAll(&buf, &gif.GIF{
		Image: images,
		Delay: delays,
	})
	if err != nil {
		return fmt.Errorf("failed to encode GIF: %v", err)
	}

	// Encode the GIF as base64 and send it to Tidbyt (only one push)
	webpBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	if err := PushToTidbyt(deviceID, webpBase64); err != nil {
		return fmt.Errorf("failed to push to Tidbyt: %v", err)
	}

	return nil
}
