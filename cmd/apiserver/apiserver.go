package apiserver

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/llmos-ai/llmos-controller/pkg/config"
	"github.com/llmos-ai/llmos-controller/pkg/server"
)

var (
	httpsPort   int
	httpPort    int
	threadiness int
	skipAuth    bool
)

func NewAPIServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apiserver",
		Short: "Run llmos-controller API server",
		RunE:  run,
	}

	cmd.PersistentFlags().IntVar(&httpsPort, "https_port", 8443, "port to listen on for https")
	cmd.PersistentFlags().IntVar(&httpPort, "http_port", 8080, "port to listen on for http")
	cmd.PersistentFlags().IntVar(&threadiness, "threadiness", 2, "number of threads to run the controller")
	cmd.PersistentFlags().BoolVar(&skipAuth, "skip_auth", false, "skip authentication")
	return cmd
}

func run(cmd *cobra.Command, _ []string) error {
	opts := server.Options{}
	opts.Context = cmd.Context()
	opts.HTTPSListenPort = httpsPort
	opts.HTTPListenPort = httpPort
	opts.Threadiness = threadiness
	opts.SkipAuth = skipAuth
	opts.KubeConfig = viper.GetString("kubeconfig")
	opts.Namespace = viper.GetString("namespace")
	opts.Debug = viper.GetBool("debug")
	opts.Trace = viper.GetBool("trace")
	opts.LogFormat = viper.GetString("log_format")
	opts.ProfilerAddress = viper.GetString("profile_address")

	config.InitLogs(opts.CommonOptions)
	config.InitProfiling(opts.ProfilerAddress)

	server, err := server.NewServer(opts)
	if err != nil {
		return err
	}
	return server.ListenAndServe(nil)
}
