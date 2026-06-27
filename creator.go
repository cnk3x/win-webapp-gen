//go:build creator

package main

import (
	"bytes"
	"embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/cnk3x/x/flagx"
	"github.com/cnk3x/x/fsx"
	"golang.org/x/image/draw"
	"golang.org/x/sys/windows"
)

var (
	//go:embed win-webapp-loader.exe
	template []byte
	_        embed.FS
)

func main() {
	var app WebApp
	var target string
	var icon string

	flagx.Var(&icon, "icon", "i", "icon file path")
	flagx.Var(&target, "target", "t", "target file path")
	flagx.Struct(&app)
	flagx.Parse()

	if app.Url == "" {
		flagx.Usage()
		os.Exit(1)
	}

	appUrl, err := url.Parse(app.Url)
	if err != nil {
		slog.Error("url mailfrom", "err", err)
		os.Exit(1)
	}

	if target == "" {
		var name = app.Name
		if app.Name != "" {
			name = app.Name + ".exe"
		}

		if name == "" {
			name = appUrl.Hostname()
		}

		if name == "" {
			name = fsx.CleanFileName(app.Url)
		}

		target, _ = filepath.Abs(name + ".exe")
	}

	targetDir := target
	if strings.HasSuffix(targetDir, ".exe") {
		targetDir = filepath.Dir(targetDir)
	}

	if app.Name == "" {
		app.Name = filepath.Base(targetDir)
	}

	if !strings.HasSuffix(target, ".exe") {
		target = filepath.Join(target, app.Name+".exe")
	}

	if icon == "" {
		icon = findIcon(targetDir, app.Name)
	}

	if err := Create(&app, target, icon); err != nil {
		slog.Error("create app", "err", err)
		os.Exit(1)
	}
	slog.Info("create app", "target", target, "url", app.Url)
}

func Create(app *WebApp, target, icon string) error {
	extra, err := json.Marshal(app)
	if err != nil {
		return err
	}

	targetTempPath := target + ".tmp"
	defer os.Remove(targetTempPath)

	if err := os.MkdirAll(filepath.Dir(targetTempPath), 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(targetTempPath, template, 0o755); err != nil {
		return err
	}

	if icon != "" {
		if err := setExeIcon(targetTempPath, icon); err != nil {
			return err
		}
	}

	info, err := os.Stat(targetTempPath)
	if err != nil {
		return err
	}

	exeSize := info.Size()
	newAppendSize := int64(len(extra) + 8)

	if err := os.Truncate(targetTempPath, exeSize+newAppendSize); err != nil {
		return err
	}

	targetTemp, err := os.OpenFile(targetTempPath, os.O_RDWR, 0)
	if err != nil {
		return err
	}

	if _, err := targetTemp.WriteAt(extra, exeSize); err != nil {
		targetTemp.Close()
		return err
	}

	magic := make([]byte, 8)
	binary.LittleEndian.PutUint32(magic[:4], uint32(len(extra)))
	binary.LittleEndian.PutUint32(magic[4:], wbapMagic)
	if _, err := targetTemp.WriteAt(magic, exeSize+int64(len(extra))); err != nil {
		targetTemp.Close()
		return err
	}

	if err := targetTemp.Sync(); err != nil {
		targetTemp.Close()
		return err
	}

	if err := targetTemp.Close(); err != nil {
		return err
	}

	return os.Rename(targetTempPath, target)
}

// setExeIcon 设置EXE图标（从PNG文件）
func setExeIcon(exePath, iconPngPath string) error {
	// 1. PNG转ICO
	icoPath, err := pngToICO(iconPngPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("生成ICON失败: %w", err)
	}
	defer os.Remove(icoPath)

	// 2. 替换EXE图标
	return replaceExeIcon(exePath, icoPath)
}

// pngToICO 将PNG转换为ICO（包含多尺寸）
func pngToICO(pngPath string) (string, error) {
	// 读取PNG
	pngFile, err := os.Open(pngPath)
	if err != nil {
		return "", err
	}
	defer pngFile.Close()

	srcImg, err := png.Decode(pngFile)
	if err != nil {
		return "", err
	}

	// 生成各尺寸PNG数据
	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	pngDataList := make([][]byte, 0, len(sizes))
	actualSizes := make([]int, 0, len(sizes))

	for _, size := range sizes {
		// 缩放
		dst := image.NewRGBA(image.Rect(0, 0, size, size))
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), srcImg, srcImg.Bounds(), draw.Over, nil)

		// 编码为PNG
		buf := new(bytes.Buffer)
		if err := png.Encode(buf, dst); err != nil {
			return "", err
		}
		pngDataList = append(pngDataList, buf.Bytes())
		actualSizes = append(actualSizes, size)
	}

	// 创建ICO文件
	icoPath := filepath.Join(os.TempDir(), "temp_icon.ico")
	icoFile, err := os.Create(icoPath)
	if err != nil {
		return "", err
	}
	defer icoFile.Close()

	// 写入ICO头
	count := len(pngDataList)
	header := []byte{0, 0, 1, 0, byte(count), 0}
	if _, err := icoFile.Write(header); err != nil {
		return "", err
	}

	// 写入目录项和数据
	offset := 6 + count*16
	entries := make([][]byte, count)

	for i, data := range pngDataList {
		size := actualSizes[i]
		icoWidth := byte(size)
		icoHeight := byte(size)
		if size == 256 {
			icoWidth = 0
			icoHeight = 0
		}

		dataLen := len(data)
		entry := []byte{
			icoWidth, icoHeight,
			0, 0, // 颜色数(0=256+), 保留
			1, 0, // 颜色平面
			32, 0, // 位深
			byte(dataLen), byte(dataLen >> 8), byte(dataLen >> 16), byte(dataLen >> 24),
			byte(offset), byte(offset >> 8), byte(offset >> 16), byte(offset >> 24),
		}
		entries[i] = entry
		offset += dataLen
	}

	for _, entry := range entries {
		if _, err := icoFile.Write(entry); err != nil {
			return "", err
		}
	}

	for _, data := range pngDataList {
		if _, err := icoFile.Write(data); err != nil {
			return "", err
		}
	}

	return icoPath, nil
}

// replaceExeIcon 替换EXE图标
func replaceExeIcon(exePath, icoPath string) error {
	// 读取ICO
	icoData, err := os.ReadFile(icoPath)
	if err != nil {
		return err
	}

	if len(icoData) < 6 {
		return fmt.Errorf("无效的ICO文件")
	}

	// 解析ICO头
	var dir struct {
		Reserved uint16
		Type     uint16
		Count    uint16
	}
	reader := bytes.NewReader(icoData)
	if err := binary.Read(reader, binary.LittleEndian, &dir); err != nil {
		return err
	}

	if dir.Reserved != 0 || dir.Type != 1 {
		return fmt.Errorf("不是有效的ICO文件")
	}

	// 读取目录项
	type entryStruct struct {
		Width       byte
		Height      byte
		ColorCount  byte
		Reserved    byte
		Planes      uint16
		BitCount    uint16
		BytesInRes  uint32
		ImageOffset uint32
	}
	entries := make([]entryStruct, dir.Count)
	for i := 0; i < int(dir.Count); i++ {
		if err := binary.Read(reader, binary.LittleEndian, &entries[i]); err != nil {
			return err
		}
	}

	// 打开EXE资源
	exeUTF16, err := windows.UTF16PtrFromString(exePath)
	if err != nil {
		return err
	}

	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	beginUpdateResource := kernel32.NewProc("BeginUpdateResourceW")
	updateResource := kernel32.NewProc("UpdateResourceW")
	endUpdateResource := kernel32.NewProc("EndUpdateResourceW")

	hUpdate, _, _ := beginUpdateResource.Call(
		uintptr(unsafe.Pointer(exeUTF16)),
		uintptr(0),
	)
	if hUpdate == 0 {
		return fmt.Errorf("打开EXE失败（可能需要管理员权限）")
	}

	// 确保资源被正确关闭
	var updateErr error
	defer func() {
		if updateErr != nil {
			endUpdateResource.Call(hUpdate, uintptr(1))
		} else {
			endUpdateResource.Call(hUpdate, uintptr(0))
		}
	}()

	// 更新所有RT_ICON
	for i, entry := range entries {
		start := int(entry.ImageOffset)
		end := start + int(entry.BytesInRes)
		if end > len(icoData) {
			updateErr = fmt.Errorf("图标数据越界")
			return updateErr
		}
		iconData := icoData[start:end]

		ret, _, _ := updateResource.Call(
			hUpdate,
			uintptr(3), // RT_ICON
			uintptr(i+1),
			uintptr(0),
			uintptr(unsafe.Pointer(&iconData[0])),
			uintptr(len(iconData)),
		)
		if ret == 0 {
			updateErr = fmt.Errorf("更新RT_ICON(%d)失败", i+1)
			return updateErr
		}
	}

	// 构建并更新RT_GROUP_ICON
	groupSize := 6 + int(dir.Count)*14
	groupData := make([]byte, groupSize)

	binary.LittleEndian.PutUint16(groupData[0:2], 0)
	binary.LittleEndian.PutUint16(groupData[2:4], 1)
	binary.LittleEndian.PutUint16(groupData[4:6], dir.Count)

	for i, entry := range entries {
		offset := 6 + i*14
		groupData[offset] = entry.Width
		groupData[offset+1] = entry.Height
		groupData[offset+2] = entry.ColorCount
		groupData[offset+3] = entry.Reserved
		binary.LittleEndian.PutUint16(groupData[offset+4:offset+6], entry.Planes)
		binary.LittleEndian.PutUint16(groupData[offset+6:offset+8], entry.BitCount)
		binary.LittleEndian.PutUint32(groupData[offset+8:offset+12], entry.BytesInRes)
		binary.LittleEndian.PutUint16(groupData[offset+12:offset+14], uint16(i+1))
	}

	ret, _, _ := updateResource.Call(
		hUpdate,
		uintptr(14), // RT_GROUP_ICON
		uintptr(1),
		uintptr(0),
		uintptr(unsafe.Pointer(&groupData[0])),
		uintptr(len(groupData)),
	)
	if ret == 0 {
		updateErr = fmt.Errorf("更新RT_GROUP_ICON失败")
		return updateErr
	}

	return nil
}

func copyFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}

	if err := dst.Sync(); err != nil {
		dst.Close()
		return err
	}

	return dst.Close()
}

func findIcon(targetDir, name string) string {
	iconPaths := []string{
		filepath.Join(targetDir, name+".png"),
		filepath.Join(targetDir, "appicon.png"),
		filepath.Join(filepath.Dir(exePath), name+".png"),
		filepath.Join(filepath.Dir(exePath), "appicon.png"),
	}
	for _, iconPath := range iconPaths {
		if stat, _ := os.Stat(iconPath); stat != nil && stat.Mode().IsRegular() {
			return iconPath
		}
	}
	return ""
}
