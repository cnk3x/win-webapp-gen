package base

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type WebApp struct {
	Name        string `json:"name,omitempty" flag:",n"`
	Url         string `json:"url,omitempty" flag:",u"`
	Width       int    `json:"width,omitempty" flag:",w"`
	Height      int    `json:"height,omitempty" flag:",h"`
	ProxyServer string `json:"proxy_server,omitempty" flag:"proxy-server,P,socks代理（ip:端口）"`
	Insecure    bool   `json:"insecure,omitempty" flag:",i,允许不安全内容（风险）"`
	Debug       bool   `json:"debug,omitempty" flag:",D,调试模式，允许右键"`
	Backend     string `json:"backend,omitempty" flag:",b,后端配置, 格式'route=/api,url=http://127.0.0.1:7993,strip=/api'"`
	Portable    bool   `json:"portable,omitempty" flag:",p,便携配置"`
}

const Magic = 0x50414257 // 'W'=0x57, 'B'=0x42, 'A'=0x41, 'P'=0x50 的小端序排列

func ReadExeExtra(fPath string, value any) (appendSize, fileSize int64, err error) {
	var file *os.File
	if file, err = os.Open(fPath); err != nil {
		err = fmt.Errorf("os.Open: %w (%s)", err, fPath)
		return
	}
	defer file.Close()

	var stat os.FileInfo
	if stat, err = file.Stat(); err != nil {
		err = fmt.Errorf("os.Stat: %w", err)
		return
	}

	fileSize = stat.Size()
	buf := make([]byte, 8)
	if _, err = file.ReadAt(buf, fileSize-8); err != nil {
		err = fmt.Errorf("read magic code: %w", err)
		return
	}

	if find := binary.LittleEndian.Uint32(buf[4:]); find != Magic {
		err = fmt.Errorf("check magic code: %d != %d", find, Magic)
		return
	}
	appendSize += 8

	if urlSize := binary.LittleEndian.Uint32(buf[:4]); urlSize > 0 {
		data := make([]byte, urlSize)
		if _, err = file.ReadAt(data, fileSize-8-int64(urlSize)); err != nil {
			err = fmt.Errorf("read extra: %w", err)
			return
		}
		appendSize += int64(urlSize)

		if err = json.Unmarshal(data, value); err != nil {
			err = fmt.Errorf("unmarshal extra: %w", err)
			return
		}
	}

	return
}

func GetExecutePath() (exePath, exeDir, exeName, exeExt string, err error) {
	exePath, err = os.Executable()
	if err != nil {
		return
	}
	exeDir, exeName, exeExt = pathSplit(exePath, true)
	return
}

func pathSplit(fPath string, windowsExeOnly bool) (dir, name, ext string) {
	dir, name = filepath.Split(fPath)
	lower := strings.ToLower(name)

	if windowsExeOnly {
		for _, pExt := range pathExt() {
			if strings.HasSuffix(lower, pExt) {
				ext = pExt
				name = name[:len(name)-len(ext)]
				break
			}
		}
	} else {
		if i := strings.LastIndexByte(name, '.'); i > -1 {
			name, ext = name[:i], name[i:]
		}
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
