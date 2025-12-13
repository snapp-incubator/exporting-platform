package application

import (
	"context"
	"exporting_platform/configs"
	"exporting_platform/internal/exporters"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	ginLogrus "github.com/toorop/gin-logrus"
)

type Application struct {
	Config *configs.Config
	Logger *log.Logger
	Router *gin.Engine
}

func NewApplication(config *configs.Config) (*Application, error) {
	app := &Application{Config: config}

	errLogger := app.registerLogger()
	if errLogger != nil {
		return nil, errLogger
	}

	app.Logger.Debug("Registering Router")
	errRouter := app.registerRouter()
	if errRouter != nil {
		return nil, errRouter
	}

	app.Logger.Debug("Registering Routes")
	app.registerRoutes()

	app.Logger.Debug("Registering Exporters")
	app.registerExporters()
	return app, nil
}

func (a *Application) registerLogger() error {
	logInstance := log.New()
	logLevel, logLevelErr := log.ParseLevel(a.Config.Exporter.LogLevel)
	if logLevelErr != nil {
		return logLevelErr
	}

	logInstance.SetLevel(logLevel)

	a.Logger = logInstance

	return nil
}

func (a *Application) registerRouter() error {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(ginLogrus.Logger(a.Logger), gin.Recovery())

	a.Router = router

	return nil
}

func (a *Application) registerRoutes() {
	a.Logger.Debug("Registering Prometheus GIN Handler")
	a.Router.GET(a.Config.Exporter.Path, prometheusGinHandler())
}

func (a *Application) registerExporters() {
	if a.Config.Harbor.Enabled {
		a.Logger.Debug("Registering Harbor")
		prometheus.MustRegister(exporters.NewHarborCollector(a.Config.Harbor.Address, a.Config.Harbor.Token, a.Config.Harbor.UseTLS))
	}

	if a.Config.Keystone.Enabled {
		for _, cloud := range a.Config.Keystone.Clouds {
			a.Logger.Debug("Registering ", cloud.OpenstackName)
			prometheus.MustRegister(exporters.NewKeystoneCollector(cloud))
		}
	}
	if a.Config.Netbox.Enabled {
    a.Logger.Debug("Registering NetBox Collector")

 exporters.StartNetboxFetcher(
    a.Config.Netbox.Address,
    a.Config.Netbox.Token,
    a.Config.Netbox.UseTLS,
    a.Config.Netbox.IgnoreTenants,
)

prometheus.MustRegister(
    exporters.NewNetBoxSnapshotCollector("/tmp/netbox.prom"),
)
}}

func (a *Application) Run(ctx context.Context) {
	srv := http.Server{
		Addr:    a.Config.Exporter.Address,
		Handler: a.Router,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.Logger.WithError(err).Fatal("Error on running router")
		}
	}()

	<-ctx.Done()
	shutdownCTX, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	if err := srv.Shutdown(shutdownCTX); err != nil {
		a.Logger.WithContext(ctx).WithError(err).Error("could not gracefully shutdown the server")
	}
	a.Logger.Info("Router successfully closed")
}

func prometheusGinHandler() gin.HandlerFunc {
	handler := promhttp.Handler()
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}
