package diff_push

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"runtime"
	"strings"

	"github.com/ddam2k/graft/internal/utils"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

type DiffPushConfig struct {
	ImageUrl string
	Push     bool
	Username string
	Password string
	FilePath string
	Platform string
	Insecure bool
}

type SkipLayerInformation struct {
	LayerDigest string
	LayerSize   int
	Comment     string
	Author      string
}

func DiffPush(config DiffPushConfig) error {
	var err error
	var diffImage v1.Image
	var diffConfigFile *v1.ConfigFile
	var configFile *v1.ConfigFile
	var image v1.Image

	imageOs := runtime.GOOS     // "linux"
	imageArch := runtime.GOARCH // "arm64"

	if len(config.Platform) != 0 {
		sp := strings.Split(config.Platform, "/")
		if len(sp) == 2 {
			imageOs = strings.TrimSpace(sp[0])
			imageArch = strings.TrimSpace(sp[1])
		}
	}

	log.Printf("OS: %s, ARCH: %s\n", imageOs, imageArch)

	if diffImage, err = crane.Load(config.FilePath); err != nil {
		return err
	}

	if diffConfigFile, err = diffImage.ConfigFile(); err != nil || diffConfigFile == nil {
		return err
	}

	if diffConfigFile.Config.Labels["graft.image.partial"] != "true" {
		return errors.New("not partial image")
	}

	baseTagName := diffConfigFile.Config.Labels["graft.image.base"]
	targetTagName := diffConfigFile.Config.Labels["graft.image.target"]

	if baseTagName == "" {
		return errors.New("not base image")
	}

	baseImageDigest := diffConfigFile.Config.Labels["graft.image.base.digest"]
	if baseImageDigest == "" {
		return errors.New("not base image digest")
	}

	baseImage, err := Pull(config, baseTagName)
	if err != nil {
		return err
	}

	if d, err := baseImage.Digest(); err == nil {
		if d.String() != baseImageDigest {
			return errors.New("base image digest not match")
		}
	}

	baseLayers, err := baseImage.Layers()
	if err != nil {
		return err
	}

	diffLayers, err := diffImage.Layers()
	if err != nil {
		return err
	}

	if cfg, err := diffImage.ConfigFile(); err != nil {
		return err
	} else {
		configFile = cfg.DeepCopy()
		configFile.History = make([]v1.History, 0)
		configFile.RootFS.DiffIDs = make([]v1.Hash, 0)
		delete(configFile.Config.Labels, "graft.image.partial")
		delete(configFile.Config.Labels, "graft.image.partial")
		delete(configFile.Config.Labels, "graft.image.base")
		delete(configFile.Config.Labels, "graft.image.target")
		delete(configFile.Config.Labels, "graft.image.base.digest")
		delete(configFile.Config.Labels, "graft.image.target.digest")

		image, err = mutate.ConfigFile(empty.Image, configFile)
		if err != nil {
			return err
		}

	}

	if layers, err := baseImage.Layers(); err == nil {
		log.Printf("layers size = %d\n", len(layers))
	}
	baseLayerIndex := 0
	diffLayerIndex := 0
	for idx, h := range diffConfigFile.History {
		log.Printf("%d. [%d] %s\n", idx, baseLayerIndex, h.CreatedBy)
		baseLayerSize := 0
		diffLayerSize := 0
		baseLayer := baseLayers[baseLayerIndex]
		if size, err := baseLayer.Size(); err == nil {
			baseLayerSize = int(size)
		}
		diffLayer := diffLayers[diffLayerIndex]
		if size, err := diffLayer.Size(); err == nil {
			diffLayerSize = int(size)
		}
		baseLayerDigest := ""
		if d, err := baseLayer.Digest(); err == nil {
			baseLayerDigest = d.String()
		}
		diffLayerDigest := ""
		if d, err := diffLayer.Digest(); err == nil {
			diffLayerDigest = d.String()
		}

		// log.Printf("%d. base size: %d, diff size: %d\n", baseLayerIndex, baseLayerSize, diffLayerSize)

		if !h.EmptyLayer {
			log.Printf("%d. from DIFF digest: %s size: %d\n", baseLayerIndex, diffLayerDigest, diffLayerSize)
			if image, err = mutate.Append(image, mutate.Addendum{
				History: v1.History{
					Author:     h.Author,
					CreatedBy:  h.CreatedBy,
					Created:    h.Created,
					EmptyLayer: h.EmptyLayer,
					Comment:    h.Comment,
				},
				Layer: diffLayer,
			}); err != nil {
				return err
			}
			baseLayerIndex++
			diffLayerIndex++
		} else {
			if h.Author == "graft.layer.skip" {
				log.Printf("%d. from BASE digest: %s size: %d\n", baseLayerIndex, baseLayerDigest, baseLayerSize)
				var skipLayerInformation SkipLayerInformation
				if err := json.Unmarshal([]byte(h.Comment), &skipLayerInformation); err != nil {
					return err
				}
				if baseLayerDigest != skipLayerInformation.LayerDigest {
					return errors.New("layer digest not match")
				}
				if baseLayerSize != skipLayerInformation.LayerSize {
					return errors.New("layer size not match")
				}
				if image, err = mutate.Append(image, mutate.Addendum{
					History: v1.History{
						Author:     skipLayerInformation.Author,
						CreatedBy:  h.CreatedBy,
						Created:    h.Created,
						EmptyLayer: h.EmptyLayer,
						Comment:    skipLayerInformation.Comment,
					},
					Layer: baseLayer,
				}); err != nil {
					return err
				}
				baseLayerIndex++
			} else {
				if cfg, err := image.ConfigFile(); err != nil {
					return err
				} else {
					cfg.History = append(cfg.History, v1.History{
						Author:     h.Author,
						CreatedBy:  h.CreatedBy,
						Created:    h.Created,
						EmptyLayer: h.EmptyLayer,
						Comment:    h.Comment,
					})
					if image, err = mutate.ConfigFile(image, cfg); err != nil {
						return err
					}
				}
			}
		}
	}

	repo, err := name.NewRepository(config.ImageUrl)
	if err != nil {
		log.Println(err)
		return err
	}

	tag := repo.Tag(targetTagName)

	repository, tagName := utils.ParseRepositoryUrl(config.ImageUrl)
	path := strings.ReplaceAll(repository, "/", "-") + "." + tagName + "." + imageOs + "." + imageArch + ".tar"

	if config.Push {

		customTransport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.Insecure,
			},
		}

		if config.Username != "" && config.Password != "" {
			if err := remote.Write(tag, image, remote.WithAuth(&authn.Basic{
				Username: config.Username,
				Password: config.Password,
			}), remote.WithTransport(customTransport)); err != nil {
				log.Println(err)
				return err
			}
		} else {
			if err := remote.Write(tag, image, remote.WithTransport(customTransport)); err != nil {
				log.Println(err)
				return err
			}
		}
	} else {
		tagToImage := map[name.Tag]v1.Image{
			tag: image,
		}

		if err := tarball.MultiWriteToFile(path, tagToImage); err != nil {
			return err
		}
	}

	return nil
}

func Pull(config DiffPushConfig, tag string) (img v1.Image, err error) {
	imageOs := runtime.GOOS     // "linux"
	imageArch := runtime.GOARCH // "arm64"
	repository := ""
	tagName := ""

	if len(config.Platform) != 0 {
		sp := strings.Split(config.Platform, "/")
		if len(sp) == 2 {
			imageOs = strings.TrimSpace(sp[0])
			imageArch = strings.TrimSpace(sp[1])
		}
	}

	repository, tagName = utils.ParseRepositoryUrl(config.ImageUrl)
	if config.FilePath == "" {
		config.FilePath = strings.ReplaceAll(repository, "/", "-") + "." + tagName + "." + imageOs + "." + imageArch + ".tar"
	}

	ref, err := name.ParseReference(repository + ":" + tag)
	if err != nil {
		log.Printf("parsing reference %q: %v", repository+":"+tag, err)
		return nil, err
	}

	customTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.Insecure,
		},
	}

	var rmt *remote.Descriptor
	if config.Username != "" && config.Password != "" {
		if rmt, err = remote.Get(ref, remote.WithPlatform(v1.Platform{Architecture: imageArch, OS: imageOs}), remote.WithAuth(&authn.Basic{
			Username: config.Username,
			Password: config.Password,
		}), remote.WithTransport(customTransport)); err != nil {
			log.Printf("remote.Get error: %v", err)
			return nil, err
		}
	} else {
		if rmt, err = remote.Get(ref, remote.WithPlatform(v1.Platform{Architecture: imageArch, OS: imageOs}), remote.WithTransport(customTransport)); err != nil {
			log.Printf("remote.Get error: %v", err)
			return nil, err
		}
	}

	img, err = rmt.Image()
	if err != nil {
		log.Printf("pulling %s: %v", config.ImageUrl, err)
		return nil, err
	}

	return img, nil
}
