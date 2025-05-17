package disaster

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

var errInvalidFileType = errors.New("invalid file type")

func uploadPhoto(fileHeader *multipart.FileHeader, baseURL string) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	if _, err = file.Read(buffer); err != nil {
		return "", err
	}

	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	fileType := http.DetectContentType(buffer)
	if !strings.HasPrefix(fileType, "image/") {
		return "", err
	}

	// TODO: Extension should depend on what the file type is, but this works for now.
	ext := "jpg"
	suffix, err := randomHex(3)
	if err != nil {
		return "", fmt.Errorf("upload photo: failed to generate random suffix: %w", err)
	}
	now := time.Now()
	fileName := fmt.Sprintf(
		"report_%s_%09d_%s.%s",
		now.Format("20060102-150405"),
		now.Nanosecond(),
		suffix, // Add random suffix just to make sure the file name is unique
		ext,
	)

	uploadDir := "_temp/photos"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return "", err
	}

	filePath := filepath.Join(uploadDir, fileName)
	fileURL := fmt.Sprintf("%s/%s", baseURL, filePath)

	dest, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		return "", err
	}

	return fileURL, nil
}
