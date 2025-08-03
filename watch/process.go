package watch

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/corona10/goimagehash"
	"github.com/koss-shtukert/bobryeye/config"
	"github.com/koss-shtukert/bobryeye/telegram"
	"github.com/rs/zerolog"
)

func getDiffBounds(img1, img2 image.Image) image.Rectangle {
	const threshold = 20

	bounds := img1.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r1, g1, b1, _ := img1.At(x, y).RGBA()
			r2, g2, b2, _ := img2.At(x, y).RGBA()

			if absDiff(r1, r2) > threshold || absDiff(g1, g2) > threshold || absDiff(b1, b2) > threshold {
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	if minX > maxX || minY > maxY {
		return image.Rect(0, 0, 0, 0)
	}
	return image.Rect(minX, minY, maxX+1, maxY+1)
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

func drawRectangle(img draw.Image, rect image.Rectangle, col color.Color) {
	for x := rect.Min.X; x < rect.Max.X; x++ {
		img.Set(x, rect.Min.Y, col)
		img.Set(x, rect.Max.Y-1, col)
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		img.Set(rect.Min.X, y, col)
		img.Set(rect.Max.X-1, y, col)
	}
}

func Process(cfg config.CameraConfig, bot *telegram.Client, log zerolog.Logger) {
	if !cfg.Enabled {
		log.Info().Str("camera", cfg.Name).Msg("Camera is disabled. Skipping.")
		return
	}

	var prevHash *goimagehash.ImageHash
	var prevImg image.Image
	var lastMotionTime time.Time
	const cooldown = 10 * time.Second

	for {
		resp, err := http.Get(cfg.SnapshotURL)
		if err != nil {
			log.Error().Err(err).Str("camera", cfg.Name).Msg("Failed to fetch snapshot")
			time.Sleep(2 * time.Second)
			continue
		}

		frame, err := jpeg.Decode(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Error().Err(err).Str("camera", cfg.Name).Msg("Failed to decode snapshot JPEG")
			time.Sleep(2 * time.Second)
			continue
		}

		hash, err := goimagehash.DifferenceHash(frame)
		if err != nil {
			log.Error().Err(err).Msg("Failed to calculate image hash")
			continue
		}

		if prevHash != nil {
			dist, _ := prevHash.Distance(hash)
			percent := float64(dist) / 64.0 * 100.0

			bounds := getDiffBounds(prevImg, frame)

			if percent >= cfg.ThresholdPercent && !bounds.Empty() && time.Since(lastMotionTime) > cooldown {
				log.Info().
					Str("camera", cfg.Name).
					Float64("change", percent).
					Str("hash", fmt.Sprintf("%s", hash.ToString())).
					Msg("Motion detected")

				filename := fmt.Sprintf("snapshot_%s_%d.jpg", cfg.Name, time.Now().Unix())
				path := filepath.Join(os.TempDir(), filename)

				rgba := image.NewRGBA(frame.Bounds())
				draw.Draw(rgba, frame.Bounds(), frame, image.Point{}, draw.Src)
				drawRectangle(rgba, bounds, color.RGBA{255, 0, 0, 255})

				f, err := os.Create(path)
				if err != nil {
					log.Error().Err(err).Msg("Failed to create snapshot file")
					continue
				}
				err = jpeg.Encode(f, rgba, nil)
				f.Close()
				if err != nil {
					log.Error().Err(err).Msg("Failed to encode JPEG with rectangle")
					continue
				}

				err = bot.SendPhoto(path, fmt.Sprintf("%s: motion detected", cfg.Name))
				if err != nil {
					log.Error().Err(err).Msg("Failed to send photo")
				} else {
					lastMotionTime = time.Now()
				}

				_ = os.Remove(path)
			}
		}

		prevHash = hash
		prevImg = frame
		time.Sleep(time.Second)
	}
}
