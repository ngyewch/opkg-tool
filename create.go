package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/urfave/cli/v2"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func doCreate(cCtx *cli.Context) error {
	if cCtx.NArg() != 2 {
		return fmt.Errorf("insufficient number of arguments")
	}
	outputFile := cCtx.Args().Get(0)
	inputDir := cCtx.Args().Get(1)

	headerCustomizer, err := newHeaderCustomizer(cCtx)
	if err != nil {
		return err
	}

	outputWriter, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	outputTarGzWriter, err := NewTarGzWriter(outputWriter, headerCustomizer)
	if err != nil {
		return err
	}
	defer func(outputTarGzWriter *TarGzWriter) {
		_ = outputTarGzWriter.Close()
	}(outputTarGzWriter)

	controlTarGzFile, err := os.CreateTemp("", "control-*.tar.gz")
	if err != nil {
		return err
	}
	controlTarGzWriter, err := NewTarGzWriter(controlTarGzFile, headerCustomizer)
	if err != nil {
		return err
	}

	dataTarGzFile, err := os.CreateTemp("", "data-*.tar.gz")
	if err != nil {
		return err
	}
	dataTarGzWriter, err := NewTarGzWriter(dataTarGzFile, headerCustomizer)
	if err != nil {
		return err
	}

	err = outputTarGzWriter.WriteBytesAsFile("./debian-binary", []byte("2.0\n"))
	if err != nil {
		return err
	}

	err = filepath.WalkDir(inputDir, func(path string, dirEntry fs.DirEntry, err error) error {
		relativePath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}
		newPath := "./" + relativePath
		if dirEntry.IsDir() {
			if newPath == "./." {
				err = dataTarGzWriter.WriteFile("./", path)
				if err != nil {
					return err
				}
				return nil
			}
			if newPath == "./CONTROL" {
				err = controlTarGzWriter.WriteFile("./", path)
				if err != nil {
					return err
				}
				return nil
			}
		}
		if strings.HasPrefix(newPath, "./CONTROL/") {
			err = controlTarGzWriter.WriteFile("./"+newPath[10:], path)
			if err != nil {
				return err
			}
		} else {
			err = dataTarGzWriter.WriteFile(newPath, path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = controlTarGzWriter.Close()
	if err != nil {
		return err
	}
	err = dataTarGzWriter.Close()
	if err != nil {
		return err
	}

	err = outputTarGzWriter.WriteFile("./control.tar.gz", controlTarGzFile.Name())
	if err != nil {
		return err
	}
	err = outputTarGzWriter.WriteFile("./data.tar.gz", dataTarGzFile.Name())
	if err != nil {
		return err
	}

	err = os.Remove(controlTarGzFile.Name())
	if err != nil {
		return err
	}
	err = os.Remove(dataTarGzFile.Name())
	if err != nil {
		return err
	}

	return nil
}

func newHeaderCustomizer(cCtx *cli.Context) (func(tarHeader *tar.Header) error, error) {
	var hasUid bool
	var hasUname bool
	var hasGid bool
	var hasGname bool
	var uid int
	var uname string
	var gid int
	var gname string

	if cCtx.IsSet(uidFlag.Name) {
		hasUid = true
		uid = uidFlag.Get(cCtx)
	}
	if cCtx.IsSet(unameFlag.Name) {
		hasUname = true
		uname = unameFlag.Get(cCtx)
	}
	if cCtx.IsSet(gidFlag.Name) {
		hasGid = true
		gid = gidFlag.Get(cCtx)
	}
	if cCtx.IsSet(gnameFlag.Name) {
		hasGname = true
		gname = gnameFlag.Get(cCtx)
	}

	if !hasUid && hasUname {
		osUser, err := user.Lookup(uname)
		if err != nil {
			return nil, err
		}
		uid, err = strconv.Atoi(osUser.Uid)
		if err != nil {
			return nil, err
		}
		hasUid = true
	} else if hasUid && !hasUname {
		osUser, err := user.LookupId(strconv.FormatInt(int64(uid), 10))
		if err != nil {
			return nil, err
		}
		uname = osUser.Username
		hasUname = true
	}

	if !hasGid && hasGname {
		osGroup, err := user.LookupGroup(gname)
		if err != nil {
			return nil, err
		}
		gid, err = strconv.Atoi(osGroup.Gid)
		if err != nil {
			return nil, err
		}
		hasGid = true
	} else if hasGid && !hasGname {
		osGroup, err := user.LookupGroupId(strconv.FormatInt(int64(gid), 10))
		if err != nil {
			return nil, err
		}
		gname = osGroup.Name
		hasGname = true
	}

	return func(tarHeader *tar.Header) error {
		if hasUid {
			tarHeader.Uid = uid
		}
		if hasUname {
			tarHeader.Uname = uname
		}
		if hasGid {
			tarHeader.Gid = gid
		}
		if hasGname {
			tarHeader.Gname = gname
		}
		return nil
	}, nil
}

type TarGzWriter struct {
	w                io.WriteCloser
	headerCustomizer func(tarHeader *tar.Header) error
	gzWriter         *gzip.Writer
	tarWriter        *tar.Writer
}

func NewTarGzWriter(w io.WriteCloser, headerCustomizer func(tarHeader *tar.Header) error) (*TarGzWriter, error) {
	gzWriter, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	tarWriter := tar.NewWriter(gzWriter)
	return &TarGzWriter{
		w:                w,
		headerCustomizer: headerCustomizer,
		gzWriter:         gzWriter,
		tarWriter:        tarWriter,
	}, nil

}

func (t *TarGzWriter) WriteBytesAsFile(name string, data []byte) error {
	header := &tar.Header{
		Name:       name,
		Size:       int64(len(data)),
		Typeflag:   tar.TypeReg,
		Mode:       0644,
		AccessTime: time.Now(),
		ChangeTime: time.Now(),
		ModTime:    time.Now(),
	}
	osUser, err := user.Current()
	if err != nil {
		return err
	}
	osGroup, err := user.LookupGroupId(osUser.Gid)
	if err != nil {
		return err
	}
	uid, err := strconv.Atoi(osUser.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(osGroup.Gid)
	if err != nil {
		return err
	}
	header.Uid = uid
	header.Gid = gid
	header.Uname = osUser.Username
	header.Gname = osGroup.Name
	err = t.headerCustomizer(header)
	if err != nil {
		return err
	}
	err = t.tarWriter.WriteHeader(header)
	if err != nil {
		return err
	}
	_, err = t.tarWriter.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (t *TarGzWriter) WriteFile(name string, path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(fileInfo, "")
	if err != nil {
		return err
	}
	header.Name = name
	err = t.headerCustomizer(header)
	if err != nil {
		return err
	}
	err = t.tarWriter.WriteHeader(header)
	if err != nil {
		return err
	}
	if fileInfo.IsDir() || (fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink) {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	_, err = io.Copy(t.tarWriter, f)
	return nil
}

func (t *TarGzWriter) Close() error {
	err := t.tarWriter.Close()
	if err != nil {
		return err
	}
	err = t.gzWriter.Close()
	if err != nil {
		return err
	}
	err = t.w.Close()
	if err != nil {
		return err
	}
	return nil
}
