package api

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"

	"rsc.io/qr"
)

// generateQRCodeBase64 generates a QR code as a base64-encoded PNG data URL.
func generateQRCodeBase64(data string, size int) (string, error) {
	if size <= 0 {
		size = 256
	}

	// Generate QR code
	code, err := qr.Encode(data, qr.L)
	if err != nil {
		return "", err
	}

	// Create image
	qrSize := code.Size
	scale := size / qrSize
	if scale < 1 {
		scale = 1
	}

	imgSize := qrSize * scale
	img := image.NewRGBA(image.Rect(0, 0, imgSize, imgSize))

	// Fill with white background
	white := color.RGBA{255, 255, 255, 255}
	black := color.RGBA{0, 0, 0, 255}

	for y := 0; y < imgSize; y++ {
		for x := 0; x < imgSize; x++ {
			img.Set(x, y, white)
		}
	}

	// Draw QR code
	for y := 0; y < qrSize; y++ {
		for x := 0; x < qrSize; x++ {
			if code.Black(x, y) {
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						img.Set(x*scale+dx, y*scale+dy, black)
					}
				}
			}
		}
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}

	// Convert to base64 data URL
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "data:image/png;base64," + b64, nil
}
