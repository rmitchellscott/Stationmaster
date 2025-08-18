package version

import "fmt"

var (
	Version   = "0.1.0"
	BuildTime = "development"
	GitCommit = "unknown"
)

func String() string {
	return fmt.Sprintf("v%s", Version)
}

func Get() map[string]string {
	return map[string]string{
		"version":   Version,
		"buildTime": BuildTime,
		"gitCommit": GitCommit,
	}
}
