module github.com/nmaupu/flux-checkver

go 1.13

require (
	github.com/Masterminds/semver v1.4.2
	github.com/fluxcd/flux v1.15.0
	github.com/gorilla/mux v1.7.3
	github.com/jawher/mow.cli v1.1.0
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/common v0.7.0
	github.com/tidwall/gjson v1.3.5 // indirect
)

replace github.com/docker/distribution => github.com/docker/distribution v2.7.1+incompatible
