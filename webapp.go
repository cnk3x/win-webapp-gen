package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
)

var exePath, exeDir, exeName, exeExt string //nolint:unused

func init() {
	exePath, _ := os.Executable()
	exeDir, exeName = filepath.Split(exePath)

	lower := strings.ToLower(exeName)
	for _, ext := range pathExt() {
		if strings.HasSuffix(lower, ext) {
			exeExt = ext
			exeName = exeName[:len(exeName)-len(exeExt)]
			break
		}
	}
}

type WebApp struct {
	Name     string `json:"name,omitempty" flag:",n"`
	Url      string `json:"url,omitempty" flag:",u"`
	Width    int    `json:"width,omitempty" flag:",w"`
	Height   int    `json:"height,omitempty" flag:",h"`
	DataPath string `json:"data_path,omitempty" flag:",d"`
	Proxy    string `json:"proxy,omitempty" flag:",p"`
	Debug    bool   `json:"debug,omitempty"`
}

const wbapMagic = 0x50414257 // 'W'=0x57, 'B'=0x42, 'A'=0x41, 'P'=0x50 的小端序排列

func readExeExtra(fPath string) (data []byte, appendSize, fileSize int64, err error) {
	var file *os.File
	if file, err = os.Open(fPath); err != nil {
		return
	}
	defer file.Close()

	var stat os.FileInfo
	if stat, err = file.Stat(); err != nil {
		return
	}

	fileSize = stat.Size()
	buf := make([]byte, 8)
	if _, err = file.ReadAt(buf, fileSize-8); err != nil {
		return
	}

	if binary.LittleEndian.Uint32(buf[4:]) != wbapMagic {
		return
	}
	appendSize += 8

	if urlSize := binary.LittleEndian.Uint32(buf[:4]); urlSize > 0 {
		data = make([]byte, urlSize)
		if _, err = file.ReadAt(data, fileSize-8-int64(urlSize)); err != nil {
			return
		}
		appendSize += int64(urlSize)
	}

	return
}

func pathExt() []string {
	var exts []string
	x := os.Getenv(`PATHEXT`)
	if x != "" {
		for e := range strings.SplitSeq(strings.ToLower(x), `;`) {
			if e == "" {
				continue
			}
			if e[0] != '.' {
				e = "." + e
			}
			exts = append(exts, e)
		}
	} else {
		exts = []string{".com", ".exe", ".bat", ".cmd"}
	}
	return exts
}
