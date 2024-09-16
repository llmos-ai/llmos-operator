package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/llmos-ai/llmos-operator/pkg/version"
)

type Version struct {
}

func NewVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "version",
		Short:              "Print operator version",
		DisableFlagParsing: true,
		RunE:               run,
	}
	return cmd
}

func run(_ *cobra.Command, _ []string) error {
	fmt.Println(version.FriendlyVersion())
	return nil
}
