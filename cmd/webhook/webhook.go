package apiserver

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/llmos-ai/llmos-controller/pkg/config"
	ws "github.com/llmos-ai/llmos-controller/pkg/webhook/server"
)

var (
	httpsPort   int
	threadiness int
	devMode     bool
	devURL      string
)

func NewWebhookServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Run llmos-controller webhook",
		RunE:  run,
	}

	cmd.PersistentFlags().IntVar(&httpsPort, "https_port", 8444, "port to listen on for https")
	cmd.PersistentFlags().IntVar(&threadiness, "threadiness", 2, "number of threads to run the controller")
	cmd.PersistentFlags().BoolVar(&devMode, "dev_mode", false, "enable local dev mode")
	cmd.PersistentFlags().StringVar(&devURL, "dev_url", "", "specify the webhook local url, only used when dev_mode is enabled")
	return cmd
}

func run(cmd *cobra.Command, _ []string) error {
	opts := ws.Options{}
	opts.Context = cmd.Context()
	opts.HTTPSListenPort = httpsPort
	opts.Threadiness = threadiness
	opts.DevMode = devMode
	opts.DevURL = devURL
	opts.KubeConfig = viper.GetString("kubeconfig")
	opts.Namespace = viper.GetString("namespace")
	opts.Debug = viper.GetBool("debug")
	opts.Trace = viper.GetBool("trace")
	opts.LogFormat = viper.GetString("log_format")
	opts.ProfilerAddress = viper.GetString("profile_address")
	opts.ReleaseName = viper.GetString("release_name")

	config.InitLogs(opts.CommonOptions)
	config.InitProfiling(opts.ProfilerAddress)

	ws, err := ws.NewServer(opts)
	if err != nil {
		return err
	}
	return ws.ListenAndServe()
}
