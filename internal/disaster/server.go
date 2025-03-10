package disaster

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/InternalPointerVariable/ResQLink-Backend/internal/api"
)

type Server struct {
	repository Repository
}

func NewServer(repository Repository) *Server {
	return &Server{
		repository: repository,
	}
}

type CivilianStatus = string

const (
	Safe     CivilianStatus = "safe"
	AtRisk   CivilianStatus = "at_risk"
	InDanger CivilianStatus = "in_danger"
)

type disasterReportResponse struct {
	DisasterReportID     string         `json:"disasterReportId"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
	Status               CivilianStatus `json:"status"`
	RawSituation         string         `json:"rawSituation"`
	AIGeneratedSituation string         `json:"aiGeneratedSituation"`
	RespondedAt          *time.Time     `json:"respondedAt"`
	UserID               string         `json:"userId"`
	PhotoURLs            []string       `json:"photoUrls"`

	// Location
	Longitude int     `json:"longitude"`
	Latitude  int     `json:"latitude"`
	Address   *string `json:"address"`
}

func (s *Server) GetDisasterReports(w http.ResponseWriter, r *http.Request) api.Response {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	reports, err := s.repository.GetDisasterReports(ctx)
	if err != nil {
		return api.Response{
			Error:   fmt.Errorf("get disaster reports: %w", err),
			Code:    http.StatusInternalServerError,
			Message: "Failed to get disaster reports.",
		}
	}

	return api.Response{
		Code:    http.StatusOK,
		Message: "Successfully fetched disaster reports.",
		Data:    reports,
	}
}

var ErrInvalidFileType = errors.New("invalid file type")

func (s *Server) CreateDisasterReport(w http.ResponseWriter, r *http.Request) api.Response {
	// ctx, cancel := context.WithCancel(r.Context())
	// defer cancel()

	const maxBodySize = 10 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	if err := r.ParseMultipartForm(maxBodySize); err != nil {
		return api.Response{
			Error:   fmt.Errorf("create disaster report: %w", err),
			Code:    http.StatusBadRequest,
			Message: "Failed to parse disaster report form data.",
		}
	}

    // TODO: Save to database 
	// var (
	// 	userID       = r.FormValue("userId")
	// 	status       = r.FormValue("status")
	// 	rawSituation = r.FormValue("rawSituation")
	// )

	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		photos := r.MultipartForm.File["photos"]

		if len(photos) > 0 {
			for _, fileHeader := range photos {
				file, err := fileHeader.Open()
				if err != nil {
					return api.Response{
						Error:   fmt.Errorf("create disaster report: %w", err),
						Code:    http.StatusInternalServerError,
						Message: "Failed to open uploaded photo.",
					}
				}
				defer file.Close()

				buffer := make([]byte, 512)
				if _, err = file.Read(buffer); err != nil {
					return api.Response{
						Error:   fmt.Errorf("create disaster report: %w", err),
						Code:    http.StatusInternalServerError,
						Message: "Failed to read file.",
					}
				}

				if _, err = file.Seek(0, io.SeekStart); err != nil {
					return api.Response{
						Error:   fmt.Errorf("create disaster report: %w", err),
						Code:    http.StatusInternalServerError,
						Message: "Failed to process file.",
					}
				}

				fileType := http.DetectContentType(buffer)
				if !strings.HasPrefix(fileType, "image/") {
					return api.Response{
						Error:   fmt.Errorf("create disaster report: %w", ErrInvalidFileType),
						Code:    http.StatusInternalServerError,
						Message: "Invalid file type.",
					}
				}

				ext := "jpg"
                // TODO: Add more details to the file name
				fileName := fmt.Sprintf("report_%s.%s", time.Now().Format("20060102150405"), ext)

				uploadDir := "_temp/photos"
				if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
					return api.Response{
						Error:   fmt.Errorf("create disaster report: %w", ErrInvalidFileType),
						Code:    http.StatusInternalServerError,
						Message: "Failed to create upload directory.",
					}
				}

				filePath := filepath.Join(uploadDir, fileName)

				dest, err := os.Create(filePath)
				if err != nil {
					return api.Response{
						Error:   fmt.Errorf("create disaster report: %w", err),
						Code:    http.StatusInternalServerError,
						Message: "Failed to create file.",
					}
				}
				defer dest.Close()

				if _, err := io.Copy(dest, file); err != nil {
					return api.Response{
						Error:   fmt.Errorf("create disaster report: %w", err),
						Code:    http.StatusInternalServerError,
						Message: "Failed to save file.",
					}
				}
			}
		}
	}

	return api.Response{
		Code:    http.StatusCreated,
		Message: "Successfully created disaster report.",
	}
}
