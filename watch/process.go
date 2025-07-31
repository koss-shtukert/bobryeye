package watch

import (
	"fmt"
	"github.com/koss-shtukert/bobryeye/config"
	"github.com/koss-shtukert/bobryeye/telegram"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"gocv.io/x/gocv"
)

func Process(cfg config.CameraConfig, bot *telegram.Client, log zerolog.Logger) {
	if !cfg.Enabled {
		log.Info().Str("camera", cfg.Name).Msg("Camera is disabled. Skipping.")
		return
	}

	log.Info().Str("camera", cfg.Name).Msg("Starting camera stream")

	webcam, err := gocv.OpenVideoCapture(cfg.RTSP)
	if err != nil {
		log.Error().Err(err).Str("camera", cfg.Name).Msg("Failed to open RTSP")
		return
	}
	defer webcam.Close()

	curr := gocv.NewMat()
	prev := gocv.NewMat()
	diff := gocv.NewMat()
	defer curr.Close()
	defer prev.Close()
	defer diff.Close()

	if ok := webcam.Read(&prev); !ok || prev.Empty() {
		log.Error().Str("camera", cfg.Name).Msg("Failed to read initial frame")
		return
	}
	gocv.CvtColor(prev, &prev, gocv.ColorBGRToGray)

	var lastMotionTime time.Time
	const cooldown = 10 * time.Second

	for {
		if ok := webcam.Read(&curr); !ok || curr.Empty() {
			time.Sleep(time.Second)
			continue
		}

		gray := gocv.NewMat()
		gocv.CvtColor(curr, &gray, gocv.ColorBGRToGray)

		gocv.AbsDiff(gray, prev, &diff)
		gocv.GaussianBlur(diff, &diff, image.Pt(5, 5), 0, 0, gocv.BorderDefault)
		gocv.Threshold(diff, &diff, 25, 255, gocv.ThresholdBinary)

		nonZero := gocv.CountNonZero(diff)
		total := diff.Rows() * diff.Cols()
		percent := float64(nonZero) / float64(total) * 100.0

		if percent >= cfg.ThresholdPercent {
			if time.Since(lastMotionTime) < cooldown {
				continue
			}

			contours := gocv.FindContours(diff, gocv.RetrievalExternal, gocv.ChainApproxSimple)

			for i := 0; i < contours.Size(); i++ {
				c := contours.At(i)
				if gocv.ContourArea(c) < 500 {
					continue
				}
				rect := gocv.BoundingRect(c)
				gocv.Rectangle(&curr, rect, color.RGBA{255, 0, 0, 0}, 2)
			}

			log.Info().Str("camera", cfg.Name).Float64("change", percent).Msg("Motion detected")

			filename := fmt.Sprintf("snapshot_%s_%d.jpg", cfg.Name, time.Now().Unix())
			path := filepath.Join(os.TempDir(), filename)

			if ok := gocv.IMWrite(path, curr); !ok {
				log.Error().Str("file", path).Msg("Failed to write snapshot")
				continue
			}

			if err := bot.SendPhoto(path, fmt.Sprintf("%s: motion detected", cfg.Name)); err != nil {
				log.Error().Err(err).Str("camera", cfg.Name).Msg("Failed to send photo")
			} else {
				lastMotionTime = time.Now()
			}

			_ = os.Remove(path)
		}

		gray.CopyTo(&prev)
	}
}
