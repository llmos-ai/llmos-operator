package version

import "fmt"

var (
	Version = "v0.0.0-dev"
	Commit  = "HEAD"
)

func FriendlyVersion() string {
	if Commit == "" {
		return Version
	}
	return fmt.Sprintf("%s-%s", Version, Commit)
}
