package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/llmos-ai/llmos-controller/cmd/apiserver"
	wServer "github.com/llmos-ai/llmos-controller/cmd/webhook"
	"github.com/llmos-ai/llmos-controller/pkg/config"
)

func New() *cobra.Command {
	opts := config.CommonOptions{}
	rootCmd := &cobra.Command{
		Use:   "llmos-controller",
		Short: "llmos-controller is a controller for LLMOS",
	}

	rootCmd.PersistentFlags().StringVar(&opts.KubeConfig, "kubeconfig", "", "kubeconfig file path")
	rootCmd.PersistentFlags().StringVar(&opts.Namespace, "namespace", "llmos-system", "namespace to deploy llmos managed resources")
	rootCmd.PersistentFlags().StringVar(&opts.ReleaseName, "release_name", "llmos-controller", "release name during the installation")
	rootCmd.PersistentFlags().BoolVar(&opts.Debug, "debug", false, "enable debug mode")
	rootCmd.PersistentFlags().BoolVar(&opts.Trace, "trace", false, "enable trace mode")
	rootCmd.PersistentFlags().StringVar(&opts.ProfilerAddress, "profile_address", "0.0.0.0:6060", "address to listen on for profiling")
	rootCmd.PersistentFlags().StringVar(&opts.LogFormat, "log_format", "text", "log format [text|json|simple]")

	_ = viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	_ = viper.BindPFlag("namespace", rootCmd.PersistentFlags().Lookup("namespace"))
	_ = viper.BindPFlag("release_name", rootCmd.PersistentFlags().Lookup("release_name"))
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindPFlag("trace", rootCmd.PersistentFlags().Lookup("trace"))
	_ = viper.BindPFlag("profile_address", rootCmd.PersistentFlags().Lookup("profile_address"))
	_ = viper.BindPFlag("log_format", rootCmd.PersistentFlags().Lookup("log_format"))

	rootCmd.AddCommand(
		apiserver.NewAPIServer(),
		wServer.NewWebhookServer(),
	)
	rootCmd.SilenceUsage = true
	rootCmd.InitDefaultHelpCmd()
	return rootCmd
}
