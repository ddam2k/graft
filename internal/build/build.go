package build

import (
	"archive/tar"
	"bytes"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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

type BuilderConfig struct {
	DockerFilePath string
	RepositoryUrl  string
	Push           bool
	Username       string
	Password       string
	PullUsername   string
	PullPassword   string
	Platform       string
	Insecure       bool
	CacheDir       string
}

type Builder struct {
	config           BuilderConfig
	parser           *Parser
	currentWorkerDir string
	image            v1.Image
}

func NewBuilder(config BuilderConfig) *Builder {
	return &Builder{
		parser: NewParser(),
		config: config,
	}
}

func (b *Builder) Build() error {
	if _, err := b.parser.Parse(b.config.DockerFilePath); err != nil {
		return err
	}

	return b.buildImage()
}

func (b *Builder) buildImage() error {
	var err error
	imageOs := "linux"          // "linux"
	imageArch := runtime.GOARCH // "arm64"

	log.Printf("OS: %s, ARCH: %s\n", imageOs, imageArch)

	customTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: b.config.Insecure,
		},
	}

	if len(b.config.Platform) != 0 {
		sp := strings.Split(b.config.Platform, "/")
		if len(sp) == 2 {
			imageOs = strings.TrimSpace(sp[0])
			imageArch = strings.TrimSpace(sp[1])
		}
	}

	repository, tagName := utils.ParseRepositoryUrl(b.config.RepositoryUrl)

	path := strings.ReplaceAll(repository+"."+tagName+".tar", "/", "-")

	log.Printf("repo: %s, tag: %s\n", repository, tagName)

	b.currentWorkerDir = "/"

	if b.parser.From == "scratch" {
		if b.image, err = mutate.ConfigFile(empty.Image, &v1.ConfigFile{
			Architecture: imageArch,
			OS:           imageOs,
			Config:       v1.Config{},
		}); err != nil {
			return err
		}
	} else {
		b.image = nil

		if b.config.CacheDir != "" {
			repository, tagName := utils.ParseRepositoryUrl(b.parser.From)
			sourcePath := filepath.Join(b.config.CacheDir, strings.ReplaceAll(repository, "/", "-")+"."+tagName+"."+imageOs+"."+imageArch+".tar")
			log.Printf("Try to load from Cache: %s\n", sourcePath)
			if b.image, err = crane.Load(sourcePath); err != nil {
				return err
			} else {
				log.Printf("On Cache: %s\n", sourcePath)
			}
		}

		if b.image == nil {
			log.Printf("Try to load from Remote: %s\b", b.parser.From)

			ref, err := name.ParseReference(b.parser.From)
			if err != nil {
				return err
			}

			var rmt *remote.Descriptor
			if b.config.PullUsername != "" && b.config.PullPassword != "" {
				if rmt, err = remote.Get(ref, remote.WithPlatform(v1.Platform{Architecture: imageArch, OS: imageOs}), remote.WithAuth(&authn.Basic{
					Username: b.config.PullUsername,
					Password: b.config.PullPassword,
				}), remote.WithTransport(customTransport)); err != nil {
					return err
				}
			} else {
				if rmt, err = remote.Get(ref, remote.WithPlatform(v1.Platform{Architecture: imageArch, OS: imageOs}), remote.WithTransport(customTransport)); err != nil {
					return err
				}
			}

			if b.image, err = rmt.Image(); err != nil {
				return err
			}
		}

		if cf, err := b.image.ConfigFile(); err != nil {
			return err
		} else {
			if b.image, err = mutate.ConfigFile(b.image, cf.DeepCopy()); err != nil {
				return err
			}

			if cf.Config.WorkingDir != "" {
				b.currentWorkerDir = cf.Config.WorkingDir
			}
		}
	}

	for idx, inst := range b.parser.GetInstructions() {
		inst.CreatedAt = time.Now()
		log.Printf("%d/%d %s\n", idx+1, len(b.parser.GetInstructions()), inst.Original)
		switch inst.Instruction {
		case "ENV":
			if b.updateConfig(inst.Original, true, func(cfg *v1.ConfigFile) error {
				cfg.Config.Env = append(cfg.Config.Env, inst.Args[0])
				return nil
			}) != nil {
				return err
			}
		case "COPY":
			if newLayer, err := b.Layer(inst.Args[0], inst.Args[1]); err == nil {
				if b.image, err = mutate.Append(b.image, mutate.Addendum{
					History: v1.History{
						CreatedBy:  inst.Original,
						Created:    v1.Time{Time: time.Now()},
						EmptyLayer: false,
					},
					Layer: newLayer,
				}); err != nil {
					return err
				}
			}
		case "WORKDIR":
			if b.updateConfig(inst.Original, true, func(cfg *v1.ConfigFile) error {
				cfg.Config.WorkingDir = inst.Args[0]
				b.currentWorkerDir = inst.Args[0]
				return nil
			}) != nil {
				return err
			}
		case "ENTRYPOINT":
			if b.updateConfig(inst.Original, true, func(cfg *v1.ConfigFile) error {
				cfg.Config.Entrypoint = inst.Args
				return nil
			}) != nil {
				return err
			}
		case "EXPOSE":
			if b.updateConfig(inst.Original, true, func(cfg *v1.ConfigFile) error {
				for _, port := range inst.Args {
					cfg.Config.ExposedPorts[port] = struct{}{}
				}
				return nil
			}) != nil {
				return err
			}
		case "CMD":
			if b.updateConfig(inst.Original, true, func(cfg *v1.ConfigFile) error {
				cfg.Config.Cmd = inst.Args
				return nil
			}) != nil {
				return err
			}
		}
	}

	repo, err := name.NewRepository(repository)
	if err != nil {
		log.Println(err)
		return err
	}

	tag := repo.Tag(tagName)

	if b.config.Push {
		if b.config.Username != "" && b.config.Password != "" {
			if err := remote.Write(tag, b.image, remote.WithAuth(&authn.Basic{
				Username: b.config.Username,
				Password: b.config.Password,
			}), remote.WithTransport(customTransport)); err != nil {
				log.Println(err)
				return err
			}
		} else {
			if err := remote.Write(tag, b.image, remote.WithTransport(customTransport)); err != nil {
				log.Println(err)
				return err
			}
		}
	} else {
		tagToImage := map[name.Tag]v1.Image{
			tag: b.image,
		}

		tarball.MultiWriteToFile(path, tagToImage)
	}

	return nil
}

func (b *Builder) Layer(src string, dst string) (v1.Layer, error) {
	buf := &bytes.Buffer{}
	w := tar.NewWriter(buf)

	nativeSrcPath := filepath.FromSlash(src)

	if fi, err := os.Stat(nativeSrcPath); err != nil {
		log.Println(err)
		return nil, err
	} else {
		dst := dst
		if dst == "." {
			dst = filepath.ToSlash(filepath.Join(b.currentWorkerDir, "/"))
		} else if strings.HasPrefix(dst, "./") {
			dst = filepath.ToSlash(filepath.Join(b.currentWorkerDir, dst))
		}
		if fi.IsDir() {
			appendDirToTar(w, nativeSrcPath, dst, 1000, 1000, 0744)
		} else {
			appendFileToTar(w, nativeSrcPath, dst, 1000, 1000, 0744)
		}
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewBuffer(buf.Bytes())), nil
	})
}

func (b *Builder) updateConfig(history string, isEmptyLayer bool, fn func(*v1.ConfigFile) error) error {
	now := time.Now()
	if cfg, err := b.image.ConfigFile(); err != nil {
		return err
	} else {
		if err := fn(cfg); err != nil {
			return err
		}
		cfg.History = append(cfg.History, v1.History{
			CreatedBy:  history,
			Created:    v1.Time{Time: now},
			EmptyLayer: isEmptyLayer,
		})
		if b.image, err = mutate.ConfigFile(b.image, cfg); err != nil {
			return err
		}
	}
	return nil
}
