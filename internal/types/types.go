package types

import v1 "github.com/google/go-containerregistry/pkg/v1"

type CopyInformation struct {
	Src string
	Dst string
}

type DockerfileDescription struct {
	From        string
	Env         []string
	Copy        []CopyInformation
	Workdir     string
	Entrypoint  []string
	Cmd         []string
	ExposePorts map[string]struct{}
}

type ChangedLayerInformation struct {
	Index           int        `json:"index"`
	PreviousDigest  string     `json:"previous_digest"`
	CurrentDigest   string     `json:"current_digest"`
	PreviousSize    int64      `json:"previous_size"`
	CurrentSize     int64      `json:"current_size"`
	PreviousHistory v1.History `json:"previous_history"`
	CurrentHistory  v1.History `json:"current_history"`
	PreviousLayer   v1.Layer   `json:"-"`
	CurrentLayer    v1.Layer   `json:"-"`
}

type DiffManifest struct {
	Repository    string `json:"repository"`
	BaseTag       string `json:"baseTag"`
	TargetTag     string `json:"targetTag"`
	ChangedLayers []ChangedLayerInformation
}
