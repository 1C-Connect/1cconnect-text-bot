package main

import (
	"connect-text-bot/bot"
	"connect-text-bot/internal/botconfig_parser"
	"connect-text-bot/internal/config"
	"connect-text-bot/internal/database"
	"connect-text-bot/internal/logger"
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/fsnotify.v1"
)

func main() {
	var (
		cnf = &config.Conf{}

		configFile = flag.String("config", "./config/config.yml", "Usage: -config=<config_file>")
		botConfig  = flag.String("bot", "./config/bot.yml", "Usage: -bot=<botConfig_file>")
		debug      = flag.Bool("debug", false, "Print debug information on stderr")
	)

	flag.Parse()

	config.GetConfig(*configFile, cnf)
	cnf.RunInDebug = *debug
	cnf.BotConfig = *botConfig

	logger.InitLogger(*debug)
	logger.Info("Application starting...")

	if *debug {
		logger.Debug("Config:", cnf)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	cache := database.ConnectInMemoryCache()
	menus := botconfig_parser.InitLevels(cnf.BotConfig)

	app := gin.Default()
	app.Use(
		config.Inject("cnf", cnf),
		database.InjectInMemoryCache("cache", cache),
		botconfig_parser.InjectLevels("menus", menus),
	)

	bot.InitHooks(app, cnf)

	srv := &http.Server{
		Addr:    cnf.Server.Listen,
		Handler: app,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen: %s\n", err)
		}
	}()

	// Следим за изменениями конфига бота.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Crit(err)
	}
	defer watcher.Close()
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Rename == fsnotify.Rename {
					if event.Name != "" {
						if err := watcher.Add(event.Name); err != nil {
							logger.Warning("Не удалось найти:", event.Name)
							watcher.Remove(event.Name)
						}
					}
					logger.Warning("При таких изменениях конфигурации рекомендуется перезагрузить бота!")
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					err = menus.UpdateLevels(cnf.BotConfig)
					if err != nil {
						logger.Warning("Не корректный конфиг бота!", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	// ищем все директории в папке
	var directories []string
	err = filepath.Walk(path.Dir(cnf.BotConfig), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			directories = append(directories, path)
		}
		return nil
	})

	// устанавливаем триггер на все папки
	for _, dir := range directories {
		if err := watcher.Add(dir); err != nil {
			logger.Crit(err)
		}
	}

	logger.Info("Application started")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT)

	quit := make(chan int)

	go func() {
		for {
			sig := <-signals
			switch sig {
			// kill -SIGHUP XXXX
			// kill -SIGINT XXXX or Ctrl+c
			case syscall.SIGHUP, syscall.SIGINT:
				logger.Info("Catch OS signal! Exiting...")

				bot.DestroyHooks(cnf)

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if err := srv.Shutdown(ctx); err != nil {
					log.Fatal("App forced to shutdown:", err)
				}

				logger.Info("Application stopped correctly!")

				quit <- 0
			default:
				logger.Warning("Unknown signal")
			}
		}
	}()

	code := <-quit

	os.Exit(code)
}
