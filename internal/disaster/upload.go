package disaster

import (
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

	ext := "jpg"
	fileName := fmt.Sprintf("disaster_report_%s.%s", time.Now().Format("20060102150405"), ext)

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
