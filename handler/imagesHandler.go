package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	flux "github.com/fluxcd/flux/pkg/api/v6"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

const (
	fluxApiImagesPath = "api/flux/v10/images"
)

var (
	_                 Handler = FluxConfig{}
	imageMetricLabels         = []string{
		"name",
		"resource_namespace",
		"resource_type",
		"resource_name",
		"repo_name",
		"current_version",
		"available_versions",
		"available_images_count",
		"created_at",
		"last_fetched",
		"most_recent_version",
		"more_recent_versions"}
	fluxImageStatusGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "flux_image_status",
			Help: "Indicates information about a deployed image and the number of versions more recent available for that image as a metric",
		},
		imageMetricLabels)
)

type FluxConfig struct {
	Url     string            `json:"url"`
	Options map[string]string `json:"options"`
}

func NewFluxConfig() FluxConfig {
	fc := FluxConfig{}
	fc.Options = make(map[string]string)
	return fc
}

func (fc FluxConfig) Handle(w http.ResponseWriter, r *http.Request) {
	var ret interface{}
	imagesStatus, err := fc.CallFluxApiImages()
	if err != nil {
		log.Errorf("An error occurred fetching Flux api: %+v\n", err)
		ret = err
	} else {
		ret = imagesStatus
	}
	json.NewEncoder(w).Encode(ret)
}

func (fc FluxConfig) CallFluxApiImages() ([]flux.ImageStatus, error) {
	// Calling flux API to retrieve images information
	apiUrl, err := url.Parse(fmt.Sprintf("%s/%s", fc.Url, fluxApiImagesPath))
	if err != nil {
		log.Errorf("Error creating URL to connect to the flux api")
		return nil, err
	}

	params, _ := url.ParseQuery(apiUrl.RawQuery)
	for key, val := range fc.Options {
		params.Add(key, val)
	}
	apiUrl.RawQuery = params.Encode()

	log.Debugf("%+v\n", apiUrl.String())

	resp, err := http.Get(apiUrl.String())
	if err != nil {
		log.Errorf("Error calling flux api")
		return nil, err
	}

	defer resp.Body.Close()

	imagesStatus := make([]flux.ImageStatus, 0)
	err = json.NewDecoder(resp.Body).Decode(&imagesStatus)
	if err != nil {
		log.Errorf("Cannot get flux response")
		return nil, err
	}

	return imagesStatus, nil
}

func (fc FluxConfig) FluxExporterRunner(interval int) {
	// Registering prometheus metric
	prometheus.MustRegister(fluxImageStatusGauge)
	go func() {
		// Main thread running once in a while to refresh metrics
		for {
			// Call Flux API and get images status
			imagesStatus, err := fc.CallFluxApiImages()
			if err != nil {
				log.Errorf("An error occurred fetching from Flux api: %+v\n", err)
			} else {
				for _, is := range imagesStatus {
					// Parsing based on https://github.com/fluxcd/flux/blob/master/pkg/resource/id.go#L50
					// Parse namespace
					toks := strings.Split(is.ID.String(), ":")
					resourceNamespace := ""
					if toks != nil && len(toks) > 0 {
						resourceNamespace = toks[0]
					}
					// Parse resource type
					resourceType := ""
					resourceName := ""
					if toks != nil && len(toks) > 1 {
						toks = strings.Split(toks[1], "/")

						if len(toks) > 0 {
							resourceType = toks[0]
						}
						if len(toks) > 1 {
							resourceName = toks[1]
						}
					}

					// Browse all containers
					for _, c := range is.Containers {
						name := c.Name
						currentVersion := c.Current.ID.Tag
						repoName := c.Current.ID.Name.String()

						// Get all available versions
						var availableVersions = make([]string, 0)
						for _, avail := range c.Available {
							// Get only semver versions
							candidatVersion := avail.ID.Tag
							candidatVersionSemver, err := semver.NewVersion(candidatVersion)
							if err == nil {
								availableVersions = append(availableVersions, candidatVersionSemver.String())
							} else {
								log.Debugf("image=%s, available candidate version %s cannot be parsed, ignoring.", name, candidatVersion)
							}
						}

						// Getting all available versions more recent than current version
						moreRecentVersions := make([]string, 0)
						moreRecentVersionsSemver := make([]*semver.Version, 0)
						mostRecentVersionStr := ""
						currentVersionSemver, err := semver.NewVersion(currentVersion)
						if err != nil {
							// our current version is not semver so we are not comparing anything
							log.Debugf("Current version %s is not semver. Ignoring available versions comparison.", currentVersion)
						} else {
							// Creating constraint to check our current version against others
							// As current version has already been checked, constraint should never fail to parse
							constraint, _ := semver.NewConstraint(fmt.Sprintf("> %s", currentVersion))

							for _, ver := range availableVersions {
								versionSemver, err := semver.NewVersion(ver)
								if err != nil {
									log.Debugf("Semver is not a valid version: %+v\n", err)
									continue
								}

								if constraint.Check(versionSemver) {
									moreRecentVersions = append(moreRecentVersions, versionSemver.String())
									// Also creating a semver slice for later use
									moreRecentVersionsSemver = append(moreRecentVersionsSemver, versionSemver)
								}
							}

							// Getting the most recent version available for this image (last sorted element)
							sort.Sort(semver.Collection(moreRecentVersionsSemver))
							if len(moreRecentVersionsSemver) > 0 {
								mostRecentVersionStr = moreRecentVersionsSemver[len(moreRecentVersionsSemver)-1].String()
							}
						}

						availableImagesCount := c.AvailableImagesCount
						newImagesCount := c.NewAvailableImagesCount

						log.Debugf("Image=%s, repoName=%s, resourceType=%s, resourceName=%s, Namespace=%s, CurrentVersion=%s, AvailableImagesCount=%d, NewImagesCount=%d\n",
							name, repoName, resourceType, resourceName, resourceNamespace, currentVersion, availableImagesCount, newImagesCount)

						labelValues := prometheus.Labels{}
						labelValues["name"] = name
						labelValues["resource_namespace"] = resourceNamespace
						labelValues["resource_type"] = resourceType
						labelValues["resource_name"] = resourceName
						labelValues["repo_name"] = repoName
						labelValues["current_version"] = currentVersionSemver.String()
						labelValues["available_versions"] = strings.Join(availableVersions, ",")
						labelValues["available_images_count"] = strconv.Itoa(availableImagesCount)
						labelValues["created_at"] = strconv.FormatInt(c.Current.CreatedAt.Unix(), 10)
						labelValues["last_fetched"] = strconv.FormatInt(c.Current.LastFetched.Unix(), 10)
						labelValues["most_recent_version"] = mostRecentVersionStr
						labelValues["more_recent_versions"] = strings.Join(moreRecentVersions, ",")

						fluxImageStatusGauge.With(labelValues).Set(float64(len(moreRecentVersions)))
					}
				}
			}

			time.Sleep(time.Duration(interval) * time.Second)
		} // for
	}()
}
