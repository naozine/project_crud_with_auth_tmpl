package version

// ビルド時に -ldflags で注入される
var (
	Version     = "dev"
	Commit      = "unknown"
	BuildDate   = "unknown"
	ProjectName = "app" // クッキー名生成用（make build でフォルダ名が注入される）
)
