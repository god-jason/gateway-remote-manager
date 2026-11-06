package filesystem

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gateway-manager/pkg/grpv1"
)

// maxReadSize 文本预览/编辑的最大读取字节数
const maxReadSize = 1024 * 1024

// Sandbox 限制文件访问根目录，规范化路径并做符号链接检查
type Sandbox struct {
	root string
}

// NewSandbox 创建沙箱
func NewSandbox(root string) *Sandbox {
	if root == "" {
		root = "/"
	}
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	return &Sandbox{root: filepath.Clean(root)}
}

// resolve 将请求路径解析到沙箱内的绝对路径
func (s *Sandbox) resolve(path string) (string, error) {
	p := path
	if !filepath.IsAbs(p) {
		p = filepath.Join(s.root, p)
	}
	p = filepath.Clean(p)

	if rel, err := filepath.Rel(s.root, p); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("路径越界: %s", path)
	}
	// 若父目录已存在，解析符号链接后再校验一次
	if resolved, err := filepath.EvalSymlinks(filepath.Dir(p)); err == nil {
		if rel, err := filepath.Rel(s.root, resolved); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("路径越界（符号链接）: %s", path)
		}
	}
	return p, nil
}

// List 列出目录
func (s *Sandbox) List(path string) ([]grpv1.FileEntry, error) {
	abs, err := s.resolve(path)
	if err != nil {
		return nil, err
	}
	dirents, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	entries := make([]grpv1.FileEntry, 0, len(dirents))
	for _, de := range dirents {
		info, err := de.Info()
		if err != nil {
			continue
		}
		entries = append(entries, entryFrom(info, filepath.Join(path, de.Name())))
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

// CreateDir 创建目录
func (s *Sandbox) CreateDir(path string) error {
	abs, err := s.resolve(path)
	if err != nil {
		return err
	}
	return os.MkdirAll(abs, 0755)
}

// Delete 删除文件或目录
func (s *Sandbox) Delete(path string) error {
	abs, err := s.resolve(path)
	if err != nil {
		return err
	}
	if abs == s.root {
		return fmt.Errorf("禁止删除根目录")
	}
	return os.RemoveAll(abs)
}

// Rename 重命名/移动
func (s *Sandbox) Rename(path, newPath string) error {
	a, err := s.resolve(path)
	if err != nil {
		return err
	}
	b, err := s.resolve(newPath)
	if err != nil {
		return err
	}
	return os.Rename(a, b)
}

// Copy 复制文件或目录
func (s *Sandbox) Copy(src, dst string) error {
	a, err := s.resolve(src)
	if err != nil {
		return err
	}
	b, err := s.resolve(dst)
	if err != nil {
		return err
	}
	return copyPath(a, b)
}

// Stat 获取条目信息
func (s *Sandbox) Stat(path string) (*grpv1.FileEntry, error) {
	abs, err := s.resolve(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	e := entryFrom(info, path)
	return &e, nil
}

// ReadText 读取文本内容（超过 maxReadSize 截断）
func (s *Sandbox) ReadText(path string) (content string, size int64, truncated bool, err error) {
	abs, err := s.resolve(path)
	if err != nil {
		return "", 0, false, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", 0, false, err
	}
	if info.IsDir() {
		return "", 0, false, fmt.Errorf("目标是目录")
	}
	f, err := os.Open(abs)
	if err != nil {
		return "", 0, false, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxReadSize))
	if err != nil {
		return "", 0, false, err
	}
	return string(data), info.Size(), info.Size() > int64(len(data)), nil
}

// WriteText 写文本文件
func (s *Sandbox) WriteText(path, content string) error {
	abs, err := s.resolve(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return err
	}
	return os.WriteFile(abs, []byte(content), 0644)
}

func entryFrom(info fs.FileInfo, path string) grpv1.FileEntry {
	return grpv1.FileEntry{
		Name:       info.Name(),
		Path:       path,
		IsDir:      info.IsDir(),
		Size:       info.Size(),
		Mode:       uint32(info.Mode().Perm()),
		Perm:       info.Mode().String(),
		ModifiedAt: info.ModTime().Unix(),
	}
}

func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}
	dirents, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, de := range dirents {
		if err := copyPath(filepath.Join(src, de.Name()), filepath.Join(dst, de.Name())); err != nil {
			return err
		}
	}
	return nil
}