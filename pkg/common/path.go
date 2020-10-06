package common

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var (
	pkgRegexp = regexp.MustCompile(`^package \w+\n$`)
)

// GetExecDir return exec dir
func GetExecDir() string {
	dir, _ := os.Executable()
	exPath := filepath.Dir(dir)
	return exPath
}

func CopyWithMainPkg(dst, src string) (err error) {
	fSrc, err := os.Open(src)
	if err != nil {
		err = fmt.Errorf("open %s file failed:%w", src, err)
		return
	}
	defer fSrc.Close()
	fDst, err := os.Create(dst)
	if err != nil {
		err = fmt.Errorf("create %s file failed:%w", dst, err)
		return
	}
	defer fDst.Close()
	r := bufio.NewReader(fSrc)
	var line string
	for err == nil {
		line, err = r.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}
		if pkgRegexp.MatchString(line) {
			line = "package main"
		}
		fDst.Write([]byte(line))
	}
	if err == io.EOF {
		err = nil
	}
	return
}

func Copy(dst, src string) (err error) {
	fSrc, err := os.Open(src)
	if err != nil {
		err = fmt.Errorf("open %s file failed:%w", src, err)
		return
	}
	defer fSrc.Close()
	fDst, err := os.Create(dst)
	if err != nil {
		err = fmt.Errorf("create %s file failed:%w", dst, err)
		return
	}
	defer fDst.Close()
	_, err = io.Copy(fDst, fSrc)
	return
}
