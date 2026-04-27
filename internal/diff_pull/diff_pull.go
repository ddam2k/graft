package diff_pull

import (
	"archive/tar"
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"mgkim.hpy.kr/graft/internal/utils"
)

type DiffPullConfig struct {
	ImageUrl  string
	BaseTag   string
	TargetTag string
	Username  string
	Password  string
	FilePath  string
	Platform  string
	Insecure  bool
}

type SkipLayerInformation struct {
	LayerDigest string
	LayerSize   int
	Comment     string
	Author      string
}

func Pull(config DiffPullConfig, tag string) (img v1.Image, err error) {
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
		log.Printf("parsing reference %q: %v", config.ImageUrl, err)
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
	} else {
		log.Printf("pulled %s:%s", config.ImageUrl, tag)
	}

	return img, nil
}

func DiffPull(config DiffPullConfig) error {
	var configFile *v1.ConfigFile
	imageOs := "linux"          // "linux"
	imageArch := runtime.GOARCH // "arm64"

	log.Printf("OS: %s, ARCH: %s\n", imageOs, imageArch)

	if len(config.Platform) != 0 {
		sp := strings.Split(config.Platform, "/")
		if len(sp) == 2 {
			imageOs = strings.TrimSpace(sp[0])
			imageArch = strings.TrimSpace(sp[1])
		}
	}

	baseImage, err := Pull(config, config.BaseTag)
	if err != nil {
		return err
	}

	targetImage, err := Pull(config, config.TargetTag)
	if err != nil {
		return err
	}

	// baseConfig, err := baseImage.ConfigFile()
	// if err != nil {
	// 	return err
	// }

	targetConfig, err := targetImage.ConfigFile()
	if err != nil {
		return err
	}

	// configFile := &v1.ConfigFile{
	// 	Architecture: imageArch,
	// 	OS:           imageOs,
	// 	Config: v1.Config{
	// 		Labels: map[string]string{},
	// 	},
	// }
	if cfg, err := targetImage.ConfigFile(); err != nil {
		return err
	} else {
		configFile = cfg.DeepCopy()
		configFile.History = make([]v1.History, 0)
		configFile.RootFS.DiffIDs = make([]v1.Hash, 0)
	}

	baseDigest := ""
	if d, err := baseImage.Digest(); err == nil {
		baseDigest = d.String()
	}

	targetDigest := ""
	if d, err := targetImage.Digest(); err == nil {
		targetDigest = d.String()
	}

	configFile.Config.Labels["graft.image.partial"] = "true"
	configFile.Config.Labels["graft.image.base"] = config.BaseTag
	configFile.Config.Labels["graft.image.target"] = config.TargetTag
	configFile.Config.Labels["graft.image.base.digest"] = baseDigest
	configFile.Config.Labels["graft.image.target.digest"] = targetDigest
	configFile.History = make([]v1.History, 0)

	image, err := mutate.ConfigFile(empty.Image, configFile)
	if err != nil {
		return err
	}

	targetLayers, err := targetImage.Layers()
	if err != nil {
		return err
	}
	baseLayers, err := baseImage.Layers()
	if err != nil {
		return err
	}

	layerIndex := 0
	for idx, h := range targetConfig.History {
		log.Printf("%d. [%d] %s\n", idx, layerIndex, h.CreatedBy)
		if !h.EmptyLayer {
			layerSize := 0
			targetLayer := targetLayers[layerIndex]
			baseLayer := baseLayers[layerIndex]
			targetDigest := ""
			if d, err := targetLayer.Digest(); err == nil {
				targetDigest = d.String()
			}
			baseDigest := ""
			if d, err := baseLayer.Digest(); err == nil {
				baseDigest = d.String()
			}
			if size, err := targetLayer.Size(); err == nil {
				layerSize = int(size)
			}

			if baseDigest == targetDigest {
				log.Printf("skipping layer %d: %s size: %d\n", layerIndex, targetDigest, layerSize)
				if cfg, err := image.ConfigFile(); err != nil {
					return err
				} else {
					comment := ""
					if bytes, err := json.Marshal(SkipLayerInformation{
						LayerDigest: targetDigest,
						LayerSize:   int(layerSize),
						Comment:     h.Comment,
						Author:      h.Author,
					}); err == nil {
						comment = string(bytes)
					}
					cfg.History = append(cfg.History, v1.History{
						Author:     "graft.layer.skip",
						CreatedBy:  h.CreatedBy,
						Created:    h.Created,
						EmptyLayer: true,
						Comment:    comment,
					})
					if image, err = mutate.ConfigFile(image, cfg); err != nil {
						return err
					}
				}
			} else {
				log.Printf("layer add %d: %s size: %d\n", layerIndex, targetDigest, layerSize)
				if image, err = mutate.Append(image, mutate.Addendum{
					History: v1.History{
						Author:     h.Author,
						CreatedBy:  h.CreatedBy,
						Created:    h.Created,
						EmptyLayer: h.EmptyLayer,
						Comment:    h.Comment,
					},
					Layer: baseLayer,
				}); err != nil {
					return err
				}
			}
			layerIndex++
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

	path := strings.ReplaceAll(config.ImageUrl+"."+config.TargetTag+".partial.tar", "/", "-")

	repo, err := name.NewRepository(config.ImageUrl)
	if err != nil {
		log.Println(err)
		return err
	}

	tag := repo.Tag(config.TargetTag)

	tagToImage := map[name.Tag]v1.Image{
		tag: image,
	}

	if err := tarball.MultiWriteToFile(path, tagToImage); err != nil {
		return err
	}

	return nil
}

func TarWriter(tarWriter *tar.Writer, dst string, bytes []byte) error {
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: dst,
		Size: int64(len(bytes)),
		Gid:  1000,
		Uid:  1000,
		Mode: int64(0744),
	}); err != nil {
		return err
	}
	if _, err := tarWriter.Write(bytes); err != nil {
		return err
	}

	return nil
}
