package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func doExtract(cCtx *cli.Context) error {
	if cCtx.NArg() != 2 {
		return fmt.Errorf("insufficient number of arguments")
	}
	inputFile := cCtx.Args().Get(0)
	outputDir := cCtx.Args().Get(1)

	f, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return err
	}

	var version string
	err = readTgz(f, func(tarHeader *tar.Header, r io.Reader) error {
		switch tarHeader.Name {
		case "./debian-binary":
			b, err := io.ReadAll(r)
			if err != nil {
				return err
			}
			version = strings.TrimSpace(string(b))
		case "./data.tar.gz":
			err = readTgz(r, func(tarHeader *tar.Header, r io.Reader) error {
				outputFile := filepath.Join(outputDir, tarHeader.Name)
				switch tarHeader.Typeflag {
				case tar.TypeDir:
					err = os.MkdirAll(outputFile, os.FileMode(tarHeader.Mode))
					if err != nil {
						return err
					}
				case tar.TypeReg:
					w, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY, os.FileMode(tarHeader.Mode))
					if err != nil {
						return err
					}
					defer func(w *os.File) {
						_ = w.Close()
					}(w)
					_, err = io.Copy(w, r)
					if err != nil {
						return err
					}
				case tar.TypeSymlink:
					err = os.Symlink(tarHeader.Linkname, outputFile)
					if err != nil {
						return err
					}
				case tar.TypeLink:
					err = os.Link(tarHeader.Linkname, outputFile)
					if err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		case "./control.tar.gz":
			controlDir := filepath.Join(outputDir, "CONTROL")
			err = os.MkdirAll(controlDir, 0755)
			if err != nil {
				return err
			}
			err = readTgz(r, func(tarHeader *tar.Header, r io.Reader) error {
				outputFile := filepath.Join(controlDir, tarHeader.Name)
				switch tarHeader.Typeflag {
				case tar.TypeDir:
					err = os.MkdirAll(outputFile, os.FileMode(tarHeader.Mode))
					if err != nil {
						return err
					}
				case tar.TypeReg:
					w, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY, os.FileMode(tarHeader.Mode))
					if err != nil {
						return err
					}
					defer func(w *os.File) {
						_ = w.Close()
					}(w)
					_, err = io.Copy(w, r)
					if err != nil {
						return err
					}
				case tar.TypeSymlink:
					err = os.Symlink(tarHeader.Linkname, outputFile)
					if err != nil {
						return err
					}
				case tar.TypeLink:
					err = os.Link(tarHeader.Linkname, outputFile)
					if err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if version != "2.0" {
		return fmt.Errorf("unsupported debian-binary version: %s", version)
	}

	return nil
}

func readTgz(r io.Reader, fileHandler func(tarHeader *tar.Header, r io.Reader) error) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		tarHeader, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		err = fileHandler(tarHeader, tr)
		if err != nil {
			return err
		}
	}
	return nil
}
