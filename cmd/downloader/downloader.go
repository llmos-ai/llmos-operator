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
	resourceType string
	namespace    string
	name         string
	outputDir    string
	threadness   int
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

	cmd.PersistentFlags().StringVar(&resourceType, "type", "", "Resource type to download (models or datasetversions)")
	cmd.PersistentFlags().StringVar(&namespace, "namespace", "", "Namespace of the resource")
	cmd.PersistentFlags().StringVar(&name, "name", "", "Name of the resource")
	cmd.PersistentFlags().StringVar(&outputDir, "output-dir", "./", "Directory to save downloaded files")
	cmd.PersistentFlags().IntVar(&threadness, "threadness", 3, "Number of threads during download files")

	_ = cmd.MarkPersistentFlagRequired("type")
	_ = cmd.MarkPersistentFlagRequired("namespace")
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

	logrus.Infof("Downloading %s '%s/%s' to directory '%s'", resourceType, namespace, name, outputDir)

	c, err := newClient(opts.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create downloader: %w", err)
	}

	if err := c.Download(ctx, resourceType, namespace, name, outputDir, threadness); err != nil {
		return fmt.Errorf("failed to download %s(%s/%s): %w", resourceType, namespace, name, err)
	}

	return nil
}
