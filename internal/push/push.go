package push

import (
	"crypto/tls"
	"log"
	"net/http"
	"runtime"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"mgkim.hpy.kr/graft/internal/utils"
)

type PushConfig struct {
	ImageUrl string
	Username string
	Password string
	FilePath string
	Platform string
	Insecure bool
}

func Push(config PushConfig) error {
	var err error
	var img v1.Image

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

	if img, err = crane.Load(config.FilePath); err != nil {
		return err
	}

	repository, tagName := utils.ParseRepositoryUrl(config.ImageUrl)

	repo, err := name.NewRepository(repository)
	if err != nil {
		log.Println(err)
		return err
	}

	customTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.Insecure,
		},
	}

	tag := repo.Tag(tagName)

	if config.Username != "" && config.Password != "" {
		if err := remote.Write(tag, img, remote.WithAuth(&authn.Basic{
			Username: config.Username,
			Password: config.Password,
		}), remote.WithTransport(customTransport)); err != nil {
			log.Println(err)
			return err
		}
	} else {
		if err := remote.Write(tag, img, remote.WithTransport(customTransport)); err != nil {
			log.Println(err)
			return err
		}
	}

	return nil
}
