package watch

import (
	"fmt"
	"image"
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

func Process(cfg config.CameraConfig, bot *telegram.Client, log zerolog.Logger) {
	if !cfg.Enabled {
		log.Info().Str("camera", cfg.Name).Msg("Camera is disabled. Skipping.")
		return
	}

	var prevHash *goimagehash.ImageHash
	var prevImg image.Image
	var lastMotionTime time.Time
	cooldown := time.Duration(cfg.Cooldown) * time.Second
	tracker := NewThresholdTracker(10, 1.15)

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

			if percent <= cfg.MinThresholdPercent {
				continue
			}

			bounds := getDiffBounds(prevImg, frame)
			dynamicThreshold := tracker.Get(cfg.Name, cfg.ThresholdPercent)

			if percent > dynamicThreshold && !bounds.Empty() && time.Since(lastMotionTime) > cooldown {
				log.Info().
					Str("camera", cfg.Name).
					Float64("change", percent).
					Float64("threshold", cfg.ThresholdPercent).
					Float64("dynamicThreshold", dynamicThreshold).
					Str("hash", fmt.Sprintf("%s", hash.ToString())).
					Msg("Motion detected")

				filename := fmt.Sprintf("snapshot_%s_%d.jpg", cfg.Name, time.Now().Unix())
				path := filepath.Join(os.TempDir(), filename)

				f, err := os.Create(path)
				if err != nil {
					log.Error().Err(err).Msg("Failed to create file")
					continue
				}
				err = jpeg.Encode(f, frame, nil)
				f.Close()
				if err != nil {
					log.Error().Err(err).Msg("Failed to encode JPEG")
					continue
				}

				err = bot.SendPhoto(path, fmt.Sprintf("%s: motion detected", cfg.Name))
				if err != nil {
					log.Error().Err(err).Msg("Failed to send photo")
				} else {
					lastMotionTime = time.Now()
				}

				_ = os.Remove(path)
				tracker.Add(cfg.Name, percent)
			}
		}

		prevHash = hash
		prevImg = frame
		time.Sleep(time.Second)
	}
}
