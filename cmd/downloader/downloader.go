package downloader

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/llmos-ai/llmos-operator/pkg/config"
)

var (
	registry   string
	name       string
	outputDir  string
	threadness int
)

type Options struct {
	ResourceType string
	Namespace    string
	Name         string
	config.CommonOptions
}

func NewDownloader() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download model or dataset files from registry",
		RunE:  run,
	}

	cmd.PersistentFlags().StringVar(&registry, "registry", "", "registry name of private registry or public registry like huggingface")
	cmd.PersistentFlags().StringVar(&name, "name", "", "model name like deepseek.ai/deekseek-r1")
	cmd.PersistentFlags().StringVar(&outputDir, "output-dir", "", "Directory to save downloaded files")
	cmd.PersistentFlags().IntVar(&threadness, "threadness", 3, "Number of threads during download files")

	_ = cmd.MarkPersistentFlagRequired("name")
	_ = cmd.MarkPersistentFlagRequired("output-dir")

	return cmd
}

func run(cmd *cobra.Command, _ []string) error {
	// Initialize common options from viper (similar to apiserver)
	opts := config.CommonOptions{
		KubeConfig:      viper.GetString("kubeconfig"),
		Namespace:       viper.GetString("namespace"),
		Debug:           viper.GetBool("debug"),
		Trace:           viper.GetBool("trace"),
		LogFormat:       viper.GetString("log_format"),
		ProfilerAddress: viper.GetString("profile_address"),
		ReleaseName:     viper.GetString("release_name"),
	}

	// Initialize logs
	config.InitLogs(opts)
	config.InitProfiling(opts.ProfilerAddress)

	// Create context
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	logrus.Infof("Downloading model %s to directory %s, registry: %s", name, outputDir, registry)

	c, err := newClient(opts.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create downloader: %w", err)
	}

	if err := c.Download(ctx, registry, name, outputDir, threadness); err != nil {
		return fmt.Errorf("failed to download %s: %w", name, err)
	}

	logrus.Infof("Downloaded model %s to directory %s", name, outputDir)

	return nil
}
