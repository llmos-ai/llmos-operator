package apiserver

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/llmos-ai/llmos-operator/pkg/config"
	"github.com/llmos-ai/llmos-operator/pkg/server"
	sconfig "github.com/llmos-ai/llmos-operator/pkg/server/config"
)

var (
	httpsPort   int
	httpPort    int
	threadiness int
)

func NewAPIServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apiserver",
		Short: "Run llmos-operator API server",
		RunE:  run,
	}

	cmd.PersistentFlags().IntVar(&httpsPort, "https_port", 8443, "port to listen on for https")
	cmd.PersistentFlags().IntVar(&httpPort, "http_port", 8080, "port to listen on for http")
	cmd.PersistentFlags().IntVar(&threadiness, "threadiness", 5, "number of threads to run the controller")
	return cmd
}

func run(cmd *cobra.Command, _ []string) error {
	opts := sconfig.Options{
		Context:         cmd.Context(),
		HTTPSListenPort: httpsPort,
		HTTPListenPort:  httpPort,
		Threadiness:     threadiness,
		CommonOptions: config.CommonOptions{
			KubeConfig:      viper.GetString("kubeconfig"),
			Namespace:       viper.GetString("namespace"),
			Debug:           viper.GetBool("debug"),
			Trace:           viper.GetBool("trace"),
			LogFormat:       viper.GetString("log_format"),
			ProfilerAddress: viper.GetString("profile_address"),
			ReleaseName:     viper.GetString("release_name"),
		},
	}

	config.InitLogs(opts.CommonOptions)
	config.InitProfiling(opts.ProfilerAddress)

	server, err := server.NewServer(opts)
	if err != nil {
		return err
	}
	return server.ListenAndServe(opts, nil)
}
