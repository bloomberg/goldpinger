// Copyright 2018 Bloomberg Finance L.P.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"strings"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"go.uber.org/zap"

	"github.com/bloomberg/goldpinger/v3/pkg/goldpinger"
	"github.com/bloomberg/goldpinger/v3/pkg/restapi/operations"
	"github.com/go-openapi/swag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//go:generate swagger generate server --target ../goldpinger --name Goldpinger --spec ../swagger.yml --exclude-main

func configureFlags(api *operations.GoldpingerAPI) {
	api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{
		swag.CommandLineOptionsGroup{
			ShortDescription: "Goldpinger Config",
			LongDescription:  "",
			Options:          &goldpinger.GoldpingerConfig,
		},
	}
}

func configureAPI(api *operations.GoldpingerAPI) http.Handler {
	// configure the api here
	api.Logger = zap.S().Infof
	api.ServeError = errors.ServeError

	api.JSONConsumer = runtime.JSONConsumer()
	api.JSONProducer = runtime.JSONProducer()

	api.PingHandler = operations.PingHandlerFunc(
		func(params operations.PingParams) middleware.Responder {
			goldpinger.CountCall("received", "ping")

			ctx, cancel := context.WithTimeout(
				params.HTTPRequest.Context(),
				time.Duration(goldpinger.GoldpingerConfig.PingTimeoutMs)*time.Millisecond,
			)
			defer cancel()

			return operations.NewPingOK().WithPayload(goldpinger.GetStats(ctx))
		})

	api.CheckServicePodsHandler = operations.CheckServicePodsHandlerFunc(
		func(params operations.CheckServicePodsParams) middleware.Responder {
			goldpinger.CountCall("received", "check")

			ctx, cancel := context.WithTimeout(
				params.HTTPRequest.Context(),
				time.Duration(goldpinger.GoldpingerConfig.CheckTimeoutMs)*time.Millisecond,
			)
			defer cancel()

			return operations.NewCheckServicePodsOK().WithPayload(goldpinger.CheckNeighbours(ctx))
		})

	api.CheckAllPodsHandler = operations.CheckAllPodsHandlerFunc(
		func(params operations.CheckAllPodsParams) middleware.Responder {
			goldpinger.CountCall("received", "check_all")

			ctx, cancel := context.WithTimeout(
				params.HTTPRequest.Context(),
				time.Duration(goldpinger.GoldpingerConfig.CheckAllTimeoutMs)*time.Millisecond,
			)
			defer cancel()

			return operations.NewCheckAllPodsOK().WithPayload(goldpinger.CheckNeighboursNeighbours(ctx))
		})

	api.HealthzHandler = operations.HealthzHandlerFunc(
		func(params operations.HealthzParams) middleware.Responder {
			goldpinger.CountCall("received", "healthz")
			healthResult := goldpinger.HealthCheck()
			if *healthResult.OK {
				return operations.NewHealthzOK().WithPayload(healthResult)
			} else {
				return operations.NewHealthzServiceUnavailable().WithPayload(healthResult)
			}
		})

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix"
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

func fileServerMiddleware(next http.Handler) http.Handler {
	zap.L().Info("Added the static middleware")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fileServer := http.FileServer(http.Dir(goldpinger.GoldpingerConfig.StaticFilePath))
		if r.URL.Path == "/" {
			http.StripPrefix("/", fileServer).ServeHTTP(w, r)
		} else if r.URL.Path == "/heatmap.png" {
			goldpinger.HeatmapHandler(w, r)
		} else if strings.HasPrefix(r.URL.Path, "/static/") {
			http.StripPrefix("/static/", fileServer).ServeHTTP(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})

}

func prometheusMetricsMiddleware(next http.Handler) http.Handler {
	zap.L().Info("Added the prometheus middleware")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			http.StripPrefix("/metrics", promhttp.Handler()).ServeHTTP(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return prometheusMetricsMiddleware(fileServerMiddleware(handler))
}
