package cmd

import (
	"log"
	"net/http"
	_ "net/http/pprof" // import pprof
	"time"

	"github.com/sirupsen/logrus"
)

func initProfiling(profileAddress string) {
	// enable profiler
	if profileAddress != "" {
		go func() {
			server := http.Server{
				Addr: profileAddress,
				// fix G114: Use of net/http apiserver function that has no support for setting timeouts (gosec)
				// refer to https://app.deepsource.com/directory/analyzers/go/issues/GO-S2114
				ReadHeaderTimeout: 10 * time.Second,
			}
			log.Println(server.ListenAndServe())
		}()
	}
}

func initLogs(debug bool) {
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
}
