package disaster

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/InternalPointerVariable/ResQLink-Backend/internal/api"
)

type Server struct {
	repository Repository
	baseURL    string
}

func NewServer(repository Repository, baseURL string) *Server {
	return &Server{
		repository: repository,
		baseURL:    baseURL,
	}
}

type CitizenStatus = string

const (
	Safe     CitizenStatus = "safe"
	AtRisk   CitizenStatus = "at_risk"
	InDanger CitizenStatus = "in_danger"
)

type disasterReportResponse struct {
	DisasterReportID     string        `json:"disasterReportId"`
	CreatedAt            time.Time     `json:"createdAt"`
	UpdatedAt            time.Time     `json:"updatedAt"`
	Status               CitizenStatus `json:"status"`
	RawSituation         string        `json:"rawSituation"`
	AIGeneratedSituation *string       `json:"aiGeneratedSituation"`
	RespondedAt          *time.Time    `json:"respondedAt"`
	UserID               string        `json:"userId"`
	PhotoURLs            []string      `json:"photoUrls"`
}

func (s *Server) GetDisasterReportsByUser(w http.ResponseWriter, r *http.Request) api.Response {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	userID := r.PathValue("userID")

	reports, err := s.repository.GetDisasterReportsByUser(ctx, userID)
	if err != nil {
		return api.Response{
			Error:   fmt.Errorf("get disaster reports by user: %w", err),
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

type createDisasterReportRequest struct {
	UserID       string        `json:"userId"`
	Status       CitizenStatus `json:"status"`
	RawSituation string        `json:"rawSituation"`
	PhotoURLs    []string      `json:"photoUrls"`
}

func (s *Server) CreateDisasterReport(w http.ResponseWriter, r *http.Request) api.Response {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	const maxBodySize = 10 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	if err := r.ParseMultipartForm(maxBodySize); err != nil {
		return api.Response{
			Error:   fmt.Errorf("create disaster report: %w", err),
			Code:    http.StatusBadRequest,
			Message: "Failed to parse disaster report form data.",
		}
	}

	disasterReport := createDisasterReportRequest{
		UserID:       r.FormValue("userId"),
		Status:       r.FormValue("status"),
		RawSituation: r.FormValue("rawSituation"),
		PhotoURLs:    []string{},
	}

	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		photos := r.MultipartForm.File["photos"]

		if len(photos) > 0 {
			for _, fileHeader := range photos {
				fileURL, err := uploadPhoto(fileHeader, s.baseURL)
				if err != nil {
					return api.Response{
						Error:   fmt.Errorf("create disaster report: %w", err),
						Code:    http.StatusInternalServerError,
						Message: "Failed to upload photo.",
					}
				}

				disasterReport.PhotoURLs = append(disasterReport.PhotoURLs, fileURL)
			}
		}
	}

	if err := s.repository.CreateDisasterReport(ctx, disasterReport); err != nil {
		return api.Response{
			Error:   fmt.Errorf("create disaster report: %w", err),
			Code:    http.StatusInternalServerError,
			Message: "Failed to create disaster report.",
		}
	}

	return api.Response{
		Code:    http.StatusCreated,
		Message: "Successfully created disaster report.",
	}
}
