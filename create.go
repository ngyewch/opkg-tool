package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/urfave/cli/v2"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func doCreate(cCtx *cli.Context) error {
	if cCtx.NArg() != 2 {
		return fmt.Errorf("insufficient number of arguments")
	}
	outputFile := cCtx.Args().Get(0)
	inputDir := cCtx.Args().Get(1)

	outputWriter, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	outputTarGzWriter, err := NewTarGzWriter(outputWriter)
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
	controlTarGzWriter, err := NewTarGzWriter(controlTarGzFile)
	if err != nil {
		return err
	}

	dataTarGzFile, err := os.CreateTemp("", "data-*.tar.gz")
	if err != nil {
		return err
	}
	dataTarGzWriter, err := NewTarGzWriter(dataTarGzFile)
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
			if (newPath == ".") || (newPath == "./CONTROL") {
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

type TarGzWriter struct {
	w         io.WriteCloser
	gzWriter  *gzip.Writer
	tarWriter *tar.Writer
}

func NewTarGzWriter(w io.WriteCloser) (*TarGzWriter, error) {
	gzWriter, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	tarWriter := tar.NewWriter(gzWriter)
	return &TarGzWriter{
		w:         w,
		gzWriter:  gzWriter,
		tarWriter: tarWriter,
	}, nil

}

func (t *TarGzWriter) WriteBytesAsFile(name string, data []byte) error {
	header := &tar.Header{
		Name:       name,
		Size:       int64(len(data)),
		Typeflag:   tar.TypeReg,
		Mode:       0644,
		Uid:        0,
		Gid:        0,
		Uname:      "root",
		Gname:      "root",
		AccessTime: time.Now(),
		ChangeTime: time.Now(),
		ModTime:    time.Now(),
	}
	err := t.tarWriter.WriteHeader(header)
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
