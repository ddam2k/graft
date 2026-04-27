package pull

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"mgkim.hpy.kr/graft/internal/utils"
)

type PullConfig struct {
	ImageUrl string
	Username string
	Password string
	FilePath string
	Platform string
	Insecure bool
}

func Pull(config PullConfig) error {
	var err error
	imageOs := runtime.GOOS     // "linux"
	imageArch := runtime.GOARCH // "arm64"

	if len(config.Platform) != 0 {
		sp := strings.Split(config.Platform, "/")
		if len(sp) == 2 {
			imageOs = strings.TrimSpace(sp[0])
			imageArch = strings.TrimSpace(sp[1])
		}
	}

	if config.FilePath == "" {
		repository, tagName := utils.ParseRepositoryUrl(config.ImageUrl)
		config.FilePath = strings.ReplaceAll(repository, "/", "-") + "." + tagName + "." + imageOs + "." + imageArch + ".tar"
	}

	ref, err := name.ParseReference(config.ImageUrl)
	if err != nil {
		log.Printf("parsing reference %q: %v", config.ImageUrl, err)
		return err
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
			return err
		}
	} else {
		if rmt, err = remote.Get(ref, remote.WithPlatform(v1.Platform{Architecture: imageArch, OS: imageOs}), remote.WithTransport(customTransport)); err != nil {
			log.Printf("remote.Get error: %v", err)
			return err
		}
	}

	img, err := rmt.Image()
	if err != nil {
		log.Printf("pulling %s: %v", config.ImageUrl, err)
		return err
	}

	tag, ok := ref.(name.Tag)
	if !ok {
		return fmt.Errorf("ref wasn't a tag or digest")
	}

	tagToImage := map[name.Tag]v1.Image{
		tag: img,
	}

	tarball.MultiWriteToFile(config.FilePath, tagToImage)

	return nil
}
