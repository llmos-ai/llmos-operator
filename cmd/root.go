package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/llmos-ai/llmos-controller/cmd/apiserver"
)

var (
	kubeconfig     string
	namespace      string
	debug          bool
	profileAddress string
)

func New() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "llmos-controller",
		Short: "llmos-controller is a controller for LLMOS",
	}

	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "kubeconfig file path")
	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", "llmos-system", "namespace to deploy llmos managed resources")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug mode")
	rootCmd.PersistentFlags().StringVar(&profileAddress, "profile_address", "0.0.0.0:6060", "address to listen on for profiling")
	_ = viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	_ = viper.BindPFlag("namespace", rootCmd.PersistentFlags().Lookup("namespace"))
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	initProfiling(profileAddress)
	initLogs(viper.GetBool("debug"))

	rootCmd.AddCommand(apiserver.NewAPIServer())
	rootCmd.SilenceUsage = true
	rootCmd.InitDefaultHelpCmd()
	return rootCmd
}
