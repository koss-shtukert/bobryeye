package main

import (
	"github.com/koss-shtukert/bobryeye/config"
	"github.com/koss-shtukert/bobryeye/logger"
	"github.com/koss-shtukert/bobryeye/telegram"
	"github.com/koss-shtukert/bobryeye/watch"

	"sync"
)

func main() {
	cfg := config.LoadFromYAML("config.yaml")
	log := logger.New()
	bot := telegram.New(cfg.TelegramToken, cfg.TelegramChatID, log)

	var wg sync.WaitGroup
	for _, cam := range cfg.Cameras {
		wg.Add(1)
		go func(c config.CameraConfig) {
			defer wg.Done()
			watch.Process(c, bot, log)
		}(cam)
	}

	wg.Wait()
}
