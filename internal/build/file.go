package build

import (
	"archive/tar"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func appendFileToTar(tarWriter *tar.Writer, src string, dst string, Gid int, Uid int, Mode os.FileMode) error {
	bytes, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	log.Printf(" - File %s %s\n", src, dst)

	_, filename := filepath.Split(src)

	if strings.HasSuffix(dst, "/") {
		dst = dst + "/" + filename
	}

	if err := tarWriter.WriteHeader(&tar.Header{
		Name: dst,
		Size: int64(len(bytes)),
		Gid:  Gid,
		Uid:  Uid,
		Mode: int64(Mode),
	}); err != nil {
		return err
	}
	if _, err := tarWriter.Write(bytes); err != nil {
		return err
	}
	return nil
}

func appendDirToTar(tarWriter *tar.Writer, src string, dst string, Gid int, Uid int, Mode os.FileMode) error {
	filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			if path != src {
				return appendDirToTar(tarWriter, path, filepath.Join(dst, info.Name()), Gid, Uid, Mode)
			}
		} else {
			return appendFileToTar(tarWriter, path, filepath.Join(dst, info.Name()), Gid, Uid, Mode)
		}
		return nil
	})
	return nil
}
