package main

const (
	AppName = "flux-checkver"
	AppDesc = "Check available versions when using Weave Flux"
)

var (
	AppVersion string
)

func main() {
	if AppVersion == "" {
		AppVersion = "master"
	}

	cli.Process(AppName, AppDesc, AppVersion)
}
