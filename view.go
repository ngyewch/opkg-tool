package main

import (
	"archive/tar"
	"fmt"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"strings"
	"time"
)

func doView(cCtx *cli.Context) error {
	if cCtx.NArg() != 1 {
		return fmt.Errorf("insufficient number of arguments")
	}
	inputFile := cCtx.Args().Get(0)

	f, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	err = readTgz(f, func(tarHeader *tar.Header, r io.Reader) error {
		printEntry(tarHeader, 0)
		switch tarHeader.Name {
		case "./data.tar.gz", "./control.tar.gz":
			err = readTgz(r, func(tarHeader *tar.Header, r io.Reader) error {
				printEntry(tarHeader, 4)
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

	return nil
}

func printEntry(tarHeader *tar.Header, indent int) {
	fmt.Print(strings.Repeat(" ", indent))

	var perms string
	if tarHeader.Typeflag == tar.TypeDir {
		perms += "d"
	} else {
		perms += "-"
	}
	var mask int64 = 0o400
	permChars := []string{"r", "w", "x"}
	for i := range 9 {
		if tarHeader.Mode&mask == mask {
			perms += permChars[i%len(permChars)]
		} else {
			perms += "-"
		}
		mask >>= 1
	}
	fmt.Print(perms)
	fmt.Print(" ")

	var owner string
	if tarHeader.Uname != "" {
		owner += tarHeader.Uname
	} else {
		owner += fmt.Sprintf("%d", tarHeader.Uid)
	}
	owner += "/"
	if tarHeader.Gname != "" {
		owner += tarHeader.Gname
	} else {
		owner += fmt.Sprintf("%d", tarHeader.Gid)
	}
	fmt.Printf("%-10s", owner)
	fmt.Print(" ")

	fmt.Printf("%10d", tarHeader.Size)
	fmt.Print(" ")

	fmt.Printf("%s", tarHeader.ModTime.Format(time.DateTime))
	fmt.Print(" ")

	fmt.Print(tarHeader.Name)
	fmt.Println()
}
