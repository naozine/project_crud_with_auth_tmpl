package version

// ビルド時に -ldflags で注入される
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)
