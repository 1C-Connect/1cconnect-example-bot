package main

import (
	"context"
	"flag"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"сonnect-companion/bot"

	"сonnect-companion/config"
	"сonnect-companion/logger"
)

var (
	cnf = &config.Conf{}

	configFile = flag.String("config", "", "Usage: -config=<config_file>")
	debug      = flag.Bool("debug", false, "Print debug information on stderr")
)

func main() {
	flag.Parse()

	config.GetConfig(*configFile, cnf)
	cnf.RunInDebug = *debug

	logger.InitLogger(*debug)
	logger.Info("Application starting...")

	if *debug {
		logger.Debug("Config:", cnf)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	app := gin.Default()
	app.Use(config.Inject(cnf))

	bot.Configure(cnf)
	bot.InitHooks(app, cnf.Line)

	srv := &http.Server{
		Addr:    cnf.Server.Listen,
		Handler: app,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen: %s\n", err)
		}
	}()

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

				bot.DestroyHooks(cnf.Line)

				ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
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