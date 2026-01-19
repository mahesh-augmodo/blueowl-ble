package hardware

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileBrowser handles the logic for reading the disk.
type FileBrowser struct {
	RootPath string // e.g. /tmp or /mnt/sdcard
}

// GetNumOfTags: Count sub-directories in RootPath
func (fb *FileBrowser) GetNumOfTags() (uint32, error) {
	entries, err := os.ReadDir(fb.RootPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			slog.Error("Folder does not exist, creating it", "path", fb.RootPath)
			if err := os.MkdirAll(fb.RootPath, 0755); err != nil {
				slog.Error("Failed to create root directory", "err", err)
				return 0, err
			}
			return 0, nil
		}

		if errors.Is(err, fs.ErrPermission) {
			slog.Error("Insufficient permissions to open folder", "path", fb.RootPath)
			return 0, err
		}

		return 0, err
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() {
			// TODO: Maybe filter for some sort of flag in folder or check for videos.
			count++
		}
	}
	return uint32(count), nil
}

// GetTagInfoByIndex: Return info for the Nth folder (Alphabetical)
func (fb *FileBrowser) GetTagInfoByIndex(idx uint32) (*TagInfo, error) {
	dirs, err := fb.getSortedDirs()
	if err != nil {
		return nil, err
	}

	if int(idx) >= len(dirs) {
		return nil, fmt.Errorf("tag index %d out of bounds (count: %d)", idx, len(dirs))
	}

	dirEntry := dirs[idx]
	tagName := dirEntry.Name()
	fullPath := filepath.Join(fb.RootPath, tagName)

	// Count files inside the this tag (only .mp4)
	files, err := os.ReadDir(fullPath)
	if err != nil {
		slog.Error("failed to get tag directory", "tag", tagName, "err", err)
		return nil, err
	}

	fileCount := 0
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".mp4") {
			fileCount++
		}
	}

	return &TagInfo{
		Name:            tagName,
		NumOfRecordings: uint32(fileCount),
	}, nil

}

// GetRecordingDetails: Return info for the Nth file in a tag (Alphabetical)
func (fb *FileBrowser) GetRecordingDetails(tag string, fileIndex uint32) (*RecordingFileInfo, error) {
	tagPath := filepath.Join(fb.RootPath, tag)

	files, err := fb.getSortedFiles(tagPath)
	if err != nil {
		return nil, fmt.Errorf("tag '%s' not found or empty", tag)
	}

	if int(fileIndex) >= len(files) {
		return nil, fmt.Errorf("files index %d out of bounds", fileIndex)
	}

	f := files[fileIndex]
	info, err := f.Info()
	if err != nil {
		return nil, err
	}

	absPath, _ := filepath.Abs(filepath.Join(tagPath, f.Name()))

	// Generate a consistent ID (CRC32 of filename)
	id := uint16(crc32.ChecksumIEEE([]byte(f.Name())))

	return &RecordingFileInfo{
		ID:       id,
		FileName: f.Name(),
		Path:     absPath,
		SizeMB:   uint32(info.Size() / 1024 / 1024),
		// Assumptions
		IMUFilePath:   strings.Replace(absPath, ".mp4", ".imu", 1),
		ThumbnailPath: strings.Replace(absPath, ".mp4", ".jpg", 1),
	}, nil
}

func (fb *FileBrowser) getSortedDirs() ([]os.DirEntry, error) {
	entries, err := os.ReadDir(fb.RootPath)
	if err != nil {
		return nil, err
	}

	var dirs []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e)
		}
	}

	// Sort alphabetically by name
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Name() < dirs[j].Name()
	})

	return dirs, nil
}

func (fb *FileBrowser) getSortedFiles(path string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var files []os.DirEntry
	for _, e := range entries {
		// Filter: Must be file AND end in .mp4
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".mp4") {
			files = append(files, e)
		}
	}

	// Sort alphabetically by name
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	return files, nil
}
