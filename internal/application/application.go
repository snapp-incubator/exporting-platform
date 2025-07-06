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
	Router *gin.Engine
}

func NewApplication(config *configs.Config) (*Application, error) {
	app := &Application{Config: config}
	app.registerRouter()
	app.registerRoutes()
	app.registerExporters()
	return app, nil
}

func (a *Application) registerRouter() {
	router := gin.New()
	router.Use(ginLogrus.Logger(log.StandardLogger()), gin.Recovery())
	a.Router = router
}

func (a *Application) registerRoutes() {
	log.Debug("Registering Routes")
	a.Router.GET(a.Config.Exporter.Path, prometheusGinHandler())
}

func (a *Application) registerExporters() {
	log.Debug("Registering Exporters")
	if a.Config.Harbor.Enabled {
		prometheus.MustRegister(exporters.NewHarborCollector(a.Config.Harbor.Address, a.Config.Harbor.Token, a.Config.Harbor.UseTLS))
	}
	if a.Config.Keystone.Enabled {
		for _, cloud := range a.Config.Keystone.CloudNames {
			prometheus.MustRegister(exporters.NewKeystoneCollector(cloud))
		}
	}
}

func (a *Application) Run(ctx context.Context) {
	srv := http.Server{
		Addr:    a.Config.Exporter.Address,
		Handler: a.Router,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Error on running router")
		}
	}()

	<-ctx.Done()
	shutdownCTX, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	if err := srv.Shutdown(shutdownCTX); err != nil {
		log.WithContext(ctx).WithError(err).Error("could not gracefully shutdown the server")
	}
	log.Debug("Router successfully closed")
}

func prometheusGinHandler() gin.HandlerFunc {
	handler := promhttp.Handler()
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}
