package disaster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Repository interface {
	ListDisasterReportsByUser(ctx context.Context, userID string) ([]disasterReportResponse, error)
	CreateDisasterReport(ctx context.Context, arg createDisasterReportRequest) error
	ListDisasterReports(ctx context.Context) ([]basicReport, error)
}

type repository struct {
	querier     *pgxpool.Pool
	redisClient *redis.Client
}

func NewRepository(querier *pgxpool.Pool, redisClient *redis.Client) Repository {
	return &repository{
		querier:     querier,
		redisClient: redisClient,
	}
}

// TODO: Ordering and filtering
func (r *repository) ListDisasterReportsByUser(
	ctx context.Context,
	userID string,
) ([]disasterReportResponse, error) {
	query := `
        SELECT 
            disaster_reports.*,
            array_agg(disaster_photos.photo_url) 
                FILTER (WHERE disaster_photos.photo_url IS NOT NULL) AS photo_urls
        FROM disaster_reports 
        LEFT JOIN disaster_photos 
            ON disaster_photos.disaster_report_id = disaster_reports.disaster_report_id
        WHERE disaster_reports.user_id = ($1)
        GROUP BY disaster_reports.disaster_report_id
    `
	rows, err := r.querier.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}

	reports, err := pgx.CollectRows(rows, pgx.RowToStructByName[disasterReportResponse])
	if err != nil {
		return nil, err
	}

	return reports, nil
}

func (r *repository) CreateDisasterReport(
	ctx context.Context,
	arg createDisasterReportRequest,
) error {
	tx, err := r.querier.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
        INSERT INTO disaster_reports (
            user_id,
            status,
            raw_situation
        )
        VALUES ($1, $2, $3)
        RETURNING disaster_report_id
    `

	var disasterReportID string

	row := tx.QueryRow(ctx, query, arg.UserID, arg.Status, arg.RawSituation)
	if err := row.Scan(&disasterReportID); err != nil {
		return err
	}

	query = `
    INSERT INTO disaster_photos (photo_url, disaster_report_id)
    VALUES ($1, $2)
    `

	for _, photoURL := range arg.PhotoURLs {
		if _, err := tx.Exec(ctx, query, photoURL, disasterReportID); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

type basicInfo struct {
	DisasterReportID string        `json:"disasterReportId"`
	CreatedAt        time.Time     `json:"createdAt"`
	UpdatedAt        time.Time     `json:"updatedAt"`
	Status           citizenStatus `json:"status"`
	RespondedAt      *time.Time    `json:"respondedAt"`
}

type userBasicInfo struct {
	UserID     string  `json:"userId"`
	FirstName  string  `json:"firstName"`
	MiddleName *string `json:"middleName"`
	LastName   string  `json:"lastName"`

	// TODO: Add avatar
}

type location struct {
	Longitude int     `json:"longitude"`
	Latitude  int     `json:"latitude"`
	Address   *string `json:"address"`
}

type basicReport struct {
	Disaster basicInfo     `json:"disaster"`
	User     userBasicInfo `json:"user"`
	Location *location     `json:"location" db:"-"`
}

func (r *repository) ListDisasterReports(ctx context.Context) ([]basicReport, error) {
	query := `
	SELECT DISTINCT ON (users.user_id)
		jsonb_build_object(
			'disasterReportId', disaster_reports.disaster_report_id,
			'createdAt', disaster_reports.created_at,
			'updatedAt', disaster_reports.updated_at,
			'status', disaster_reports.status,
			'respondedAt', disaster_reports.responded_at
		) AS disaster,
		jsonb_build_object(
			'userId', users.user_id,
			'firstName', users.first_name,
			'middleName', users.middle_name,
			'lastName', users.last_name
		) AS user
	FROM disaster_reports
	JOIN users ON users.user_id = disaster_reports.user_id
	ORDER BY users.user_id,
		CASE disaster_reports.status
			WHEN 'in_danger' THEN 1
			WHEN 'at_risk' THEN 2
			WHEN 'safe' THEN 3
			ELSE 4 -- For unexpected status values
		END
	`

	rows, err := r.querier.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	reports, err := pgx.CollectRows(rows, pgx.RowToStructByName[basicReport])
	if err != nil {
		return nil, err
	}

	pipe := r.redisClient.Pipeline()
	cmds := make(map[string]*redis.JSONCmd)

	for _, report := range reports {
		key := fmt.Sprintf("user:%s:location", report.User.UserID)
		cmds[report.User.UserID] = pipe.JSONGet(ctx, key)
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	for i := range reports {
		report := &reports[i]
		result, err := cmds[report.User.UserID].Result()
		if err != nil {
			return nil, err
		}

		if result == "" {
			continue
		}

		if err := json.Unmarshal([]byte(result), &report.Location); err != nil {
			return nil, err
		}
	}

	return reports, nil
}
