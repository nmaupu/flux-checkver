package cli

import (
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/common/log"

	"github.com/gorilla/mux"
	cli "github.com/jawher/mow.cli"
	"github.com/nmaupu/flux-checkver/handler"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	appInfo    handler.AppInfo
	fluxConfig handler.FluxConfig = handler.NewFluxConfig()
	health                        = handler.Health{Status: "ok"}
)

func Process(appName, appDesc, appVersion string) {
	app := cli.App(appName, appDesc)
	appInfo.AppName = appName
	appInfo.AppDesc = appDesc
	appInfo.AppVersion = appVersion

	var (
		url       = app.StringOpt("f fluxAddress", "http://localhost:3030", "Flux API address to query")
		namespace = app.StringOpt("n namespace", "", "Get images information only for a specific namespace")
		logLevel  = app.StringOpt("l log-level", "info", "Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]")
	)

	app.Version("v version", fmt.Sprintf("%s version %s", appName, appVersion))

	app.Before = func() {
		fluxConfig.Url = *url
		fluxConfig.Options["namespace"] = *namespace
		err := log.Base().SetLevel(*logLevel)
		if err != nil {
			log.Warnf("%+v, default to info\n", err)
			log.Base().SetLevel("info")
		}
	}

	app.Command("server", "Listen for incoming http request and serve responses as JSON", server)

	app.Run(os.Args)
}

func server(cmd *cli.Cmd) {
	cmd.Spec = "[-b] [-p] [-i]"
	var (
		bind     = cmd.StringOpt("b bind", "", "Address to bind")
		port     = cmd.IntOpt("p port", 8080, "Port to bind")
		interval = cmd.IntOpt("i interval", 3600, "Specifies at which interval (seconds) Flux api is being called to refresh images' data")
	)

	cmd.Before = func() {
		// Run prometheus exporter go routine
		fluxConfig.FluxExporterRunner(*interval)
	}

	cmd.Action = func() {
		router := mux.NewRouter().StrictSlash(true)

		router.HandleFunc("/version", appInfo.Handle)
		router.HandleFunc("/health", health.Handle)
		router.HandleFunc("/images", fluxConfig.Handle)
		router.Handle("/metrics", promhttp.Handler())

		log.Infof("Listening on %s:%d\n", *bind, *port)
		err := http.ListenAndServe(fmt.Sprintf("%s:%d", *bind, *port), router)
		if err != nil {
			log.Fatalf("Error listening: %+v\n", err.Error())
			os.Exit(1)
		}
	}
}
