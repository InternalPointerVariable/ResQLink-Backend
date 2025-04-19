package disaster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/InternalPointerVariable/ResQLink-Backend/internal/user"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Repository interface {
	ListDisasterReportsByUser(ctx context.Context, userID string) (userReports, error)
	CreateDisasterReport(ctx context.Context, arg createDisasterReportRequest) error
	ListDisasterReports(ctx context.Context) ([]latestReport, error)
	SaveLocation(ctx context.Context, arg saveLocationRequest) error
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

type fullReport struct {
	basicReport

	RawSituation         string   `json:"rawSituation"`
	AIGeneratedSituation *string  `json:"aiGeneratedSituation"`
	PhotoURLs            []string `json:"photoUrls"`
}

type userReports struct {
	Reports    []fullReport   `json:"reports"`
	ReportedBy user.BasicInfo `json:"reportedBy"`
	Location   *location      `json:"location"   db:"-"`
}

// TODO: Ordering and filtering
func (r *repository) ListDisasterReportsByUser(
	ctx context.Context,
	userID string,
) (userReports, error) {
	query := `
		WITH photos AS (
			SELECT disaster_report_id,
				array_agg(photo_url) 
					FILTER (WHERE photo_url IS NOT NULL) AS photo_urls
			FROM disaster_photos
			GROUP BY disaster_report_id
		), 
		user_reports AS (
			SELECT 
				disaster_reports.user_id,
				jsonb_agg(
					jsonb_build_object(
						'disasterReportId', disaster_reports.disaster_report_id,
						'createdAt', disaster_reports.created_at,
						'updatedAt', disaster_reports.updated_at,
						'status', disaster_reports.status,
						'respondedAt', disaster_reports.responded_at,
						'rawSituation', disaster_reports.raw_situation,
						'aiGeneratedSituation', disaster_reports.ai_generated_situation,
						'photoUrls', photos.photo_urls
					)
					ORDER BY disaster_reports.created_at DESC
				) AS reports
			FROM disaster_reports 
			LEFT JOIN photos 
				ON photos.disaster_report_id = disaster_reports.disaster_report_id
			GROUP BY disaster_reports.user_id
		)
        SELECT 
			user_reports.reports,
			jsonb_build_object(
				'userId', users.user_id,
				'firstName', users.first_name,
				'middleName', users.middle_name,
				'lastName', users.last_name
			) AS reported_by
        FROM user_reports 
		JOIN users ON users.user_id = user_reports.user_id
        WHERE user_reports.user_id = ($1)
    `
	rows, err := r.querier.Query(ctx, query, userID)
	if err != nil {
		return userReports{}, err
	}

	disaster, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[userReports])
	if err != nil {
		return userReports{}, err
	}

	key := fmt.Sprintf("user:%s:location", userID)
	result, err := r.redisClient.JSONGet(ctx, key).Result()
	if err != nil {
		return userReports{}, err
	}

	if result != "" {
		if err := json.Unmarshal([]byte(result), &disaster.Location); err != nil {
			return userReports{}, err
		}
	}

	return disaster, nil
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

	argB, err := json.Marshal(arg)
	if err != nil {
		return err
	}

	if err := r.redisClient.Publish(ctx, createReport, argB).Err(); err != nil {
		return err
	}

	return nil
}

type citizenStatus = string

const (
	safe     citizenStatus = "safe"
	atRisk   citizenStatus = "at_risk"
	inDanger citizenStatus = "in_danger"
)

type basicReport struct {
	DisasterReportID string        `json:"disasterReportId"`
	CreatedAt        time.Time     `json:"createdAt"`
	UpdatedAt        time.Time     `json:"updatedAt"`
	Status           citizenStatus `json:"status"`
	RespondedAt      *time.Time    `json:"respondedAt"`
}

type location struct {
	Longitude int     `json:"longitude"`
	Latitude  int     `json:"latitude"`
	Address   *string `json:"address"`
}

type latestReport struct {
	Report     basicReport    `json:"report"`
	ReportedBy user.BasicInfo `json:"reportedBy"`
	Location   *location      `json:"location"   db:"-"`
}

func (r *repository) ListDisasterReports(ctx context.Context) ([]latestReport, error) {
	query := `
	SELECT DISTINCT ON (users.user_id)
		jsonb_build_object(
			'disasterReportId', disaster_reports.disaster_report_id,
			'createdAt', disaster_reports.created_at,
			'updatedAt', disaster_reports.updated_at,
			'status', disaster_reports.status,
			'respondedAt', disaster_reports.responded_at
		) AS report,
		jsonb_build_object(
			'userId', users.user_id,
			'firstName', users.first_name,
			'middleName', users.middle_name,
			'lastName', users.last_name
		) AS reported_by
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

	reports, err := pgx.CollectRows(rows, pgx.RowToStructByName[latestReport])
	if err != nil {
		return nil, err
	}

	pipe := r.redisClient.Pipeline()
	cmds := make(map[string]*redis.JSONCmd)

	for _, report := range reports {
		key := fmt.Sprintf("user:%s:location", report.ReportedBy.UserID)
		cmds[report.ReportedBy.UserID] = pipe.JSONGet(ctx, key)
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	for i := range reports {
		report := &reports[i]
		result, err := cmds[report.ReportedBy.UserID].Result()
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

type saveLocationRequest struct {
	Location location `json:"location"`
	UserID   string   `json:"userId"`
}

func (r *repository) SaveLocation(ctx context.Context, arg saveLocationRequest) error {
	key := fmt.Sprintf("user:%s:location", arg.UserID)
	if err := r.redisClient.JSONSet(ctx, key, "$", arg.Location).Err(); err != nil {
		return err
	}

	return nil
}
