package config

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/ehazlett/simplelog"
	"github.com/sirupsen/logrus"
)

type CommonOptions struct {
	Debug     bool
	Trace     bool
	LogFormat string

	ProfilerAddress string
	KubeConfig      string
	Namespace       string
	ReleaseName     string
	InitPassword    string
}

func InitProfiling(profileAddress string) {
	// enable profiler
	if profileAddress != "" {
		logrus.Debugf("Starting profiler on: %s", profileAddress)
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

func InitLogs(opts CommonOptions) {
	switch opts.LogFormat {
	case "simple":
		logrus.SetFormatter(&simplelog.StandardFormatter{})
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		logrus.SetFormatter(&logrus.TextFormatter{})
	}
	logrus.SetOutput(os.Stdout)
	if opts.Debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debugf("Loglevel set to [%v]", logrus.DebugLevel)
	}
	if opts.Trace {
		logrus.SetLevel(logrus.TraceLevel)
		logrus.Tracef("Loglevel set to [%v]", logrus.TraceLevel)
	}
}
