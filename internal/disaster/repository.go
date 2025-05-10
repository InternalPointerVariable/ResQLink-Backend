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
	CreateDisasterReport(ctx context.Context, arg createReportRequest) error
	ListDisasterReports(ctx context.Context) ([]basicReport, error)
	ListDisasterReportsByReporter(ctx context.Context, reporterID string) (reportsByReporterResponse, error)
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

type citizenStatus string

const (
	safe     citizenStatus = "safe"
	atRisk   citizenStatus = "at_risk"
	inDanger citizenStatus = "in_danger"
)

type reporter struct {
	ReporterID string    `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	Name       string    `json:"name"`
}

type responder struct {
	ResponderID string    `json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	Name        string    `json:"name"`
}

type location struct {
	Longitude int     `json:"longitude"`
	Latitude  int     `json:"latitude"`
	Address   *string `json:"address"`
}

type basicReport struct {
	DisasterReportID string        `json:"id"`
	CreatedAt        time.Time     `json:"createdAt"`
	UpdatedAt        time.Time     `json:"updatedAt"`
	Status           citizenStatus `json:"status"`
	Reporter         reporter      `json:"reporter"`
	Responder        *responder    `json:"responder"`
	Location         location      `json:"location"         db:"-"`
}

type fullReport struct {
	basicReport

	RawSituation   string   `json:"rawSituation"`
	AIGenSituation *string  `json:"aiGenSituation"`
	PhotoURLs      []string `json:"photoUrls"`
}

const locationFmt = "reporter:%s:location"

func (r *repository) ListDisasterReports(ctx context.Context) ([]basicReport, error) {
	query := `
	SELECT DISTINCT ON (disaster_reports.reporter_id)
		disaster_reports.disaster_report_id,
		disaster_reports.created_at,
		disaster_reports.updated_at,
		disaster_reports.status,
		jsonb_build_object(
			'id', reporters.reporter_id,
			'createdAt', reporters.created_at,
			'name', COALESCE(
				TRIM(CONCAT(users.last_name, ', ', users.first_name, ' ', users.middle_name)), 
				reporters.name
			)
		) AS reporter,
 		CASE WHEN responders.responder_id IS NOT NULL THEN
			jsonb_build_object(
				'id', responders.responder_id,
				'createdAt', responders.created_at,
				'name', COALESCE(
					TRIM(CONCAT(users.last_name, ', ', users.first_name, ' ', users.middle_name)), 
					responders.name
				)
			)
		ELSE NULL
		END AS responder
	FROM disaster_reports
	JOIN reporters ON reporters.reporter_id = disaster_reports.reporter_id
	LEFT JOIN responders ON responders.responder_id = disaster_reports.responder_id
	LEFT JOIN users ON users.user_id = reporters.user_id
	ORDER BY 
		disaster_reports.reporter_id,
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
		key := fmt.Sprintf(locationFmt, report.Reporter.ReporterID)
		cmds[report.Reporter.ReporterID] = pipe.JSONGet(ctx, key)
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	for i := range reports {
		report := &reports[i]
		result, err := cmds[report.Reporter.ReporterID].Result()
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

type userReport struct {
	DisasterReportID string        `json:"id"`
	CreatedAt        time.Time     `json:"createdAt"`
	UpdatedAt        time.Time     `json:"updatedAt"`
	Status           citizenStatus `json:"status"`
	Responder        *responder    `json:"responder"`
	RawSituation     string        `json:"rawSituation"`
	AIGenSituation   *string       `json:"aiGenSituation"`
	PhotoURLs        []string      `json:"photoUrls"`
}

type reportsByReporterResponse struct {
	Reports  []userReport `json:"reports"`
	Reporter reporter     `json:"reporter"`
	Location *location    `json:"location" db:"-"`
}

// TODO: Ordering and filtering
func (r *repository) ListDisasterReportsByReporter(
	ctx context.Context,
	reporterID string,
) (reportsByReporterResponse, error) {
	query := `
	WITH photos AS (
		SELECT 
			disaster_report_id,
			array_agg(photo_url) FILTER (WHERE photo_url IS NOT NULL) AS photo_urls
		FROM disaster_photos
		GROUP BY disaster_report_id
	),
	user_reports AS (
		SELECT 
			jsonb_agg(
				jsonb_build_object(
					'id', disaster_reports.disaster_report_id,
					'createdAt', disaster_reports.created_at,
					'updatedAt', disaster_reports.updated_at,
					'status', disaster_reports.status,
					'rawSituation', disaster_reports.raw_situation,
					'aiGenSituation', disaster_reports.ai_gen_situation,
					'photoUrls', photos.photo_urls,
					'responder', CASE WHEN responders.responder_id IS NOT NULL THEN
						jsonb_build_object(
							'id', responders.responder_id,
							'name', COALESCE(
								TRIM(CONCAT(users.last_name, ', ', users.first_name, ' ', users.middle_name)), 
								responders.name
							),
							'createdAt', responders.created_at
						)
						ELSE NULL END
				)
				ORDER BY disaster_reports.created_at DESC
			) AS reports,
			disaster_reports.reporter_id
		FROM disaster_reports 
		LEFT JOIN responders ON responders.responder_id = disaster_reports.responder_id
		LEFT JOIN users ON users.user_id = responders.user_id
		LEFT JOIN photos ON photos.disaster_report_id = disaster_reports.disaster_report_id
		GROUP BY disaster_reports.reporter_id
	)
	SELECT 
		user_reports.reports,
		jsonb_build_object(
			'id', reporters.reporter_id,
			'createdAt', reporters.created_at,
			'name', COALESCE(
				TRIM(CONCAT(users.last_name, ', ', users.first_name, ' ', users.middle_name)), 
				reporters.name
			)
		) AS reporter
	FROM user_reports 
	JOIN reporters ON reporters.reporter_id = user_reports.reporter_id
	LEFT JOIN users ON users.user_id = reporters.user_id
	WHERE user_reports.reporter_id = ($1)
	`
	rows, err := r.querier.Query(ctx, query, reporterID)
	if err != nil {
		return reportsByReporterResponse{}, err
	}

	disaster, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[reportsByReporterResponse])
	if err != nil {
		return reportsByReporterResponse{}, err
	}

	key := fmt.Sprintf(locationFmt, reporterID)
	result, err := r.redisClient.JSONGet(ctx, key).Result()
	if err != nil {
		return reportsByReporterResponse{}, err
	}

	if result != "" {
		if err := json.Unmarshal([]byte(result), &disaster.Location); err != nil {
			return reportsByReporterResponse{}, err
		}
	}

	return disaster, nil
}

func (r *repository) CreateDisasterReport(
	ctx context.Context,
	arg createReportRequest,
) error {
	tx, err := r.querier.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
	INSERT INTO reporters (name, user_id)
	VALUES ($1, $2)
	ON CONFLICT (user_id) DO UPDATE
		SET name = EXCLUDED.name
	RETURNING reporter_id
	`

	var reporterID string

	row := tx.QueryRow(ctx, query, arg.Name, arg.UserID)
	if err := row.Scan(&reporterID); err != nil {
		return err
	}

	query = `
        INSERT INTO disaster_reports (
            status,
            raw_situation,
            reporter_id
        )
        VALUES ($1, $2, $3)
        RETURNING disaster_report_id
    `

	var disasterReportID string

	row = tx.QueryRow(ctx, query, arg.Status, arg.RawSituation, reporterID)
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

type saveLocationRequest struct {
	Location   location `json:"location"`
	ReporterID string   `json:"reporterId"`
}

func (r *repository) SaveLocation(ctx context.Context, arg saveLocationRequest) error {
	key := fmt.Sprintf(locationFmt, arg.ReporterID)
	if err := r.redisClient.JSONSet(ctx, key, "$", arg.Location).Err(); err != nil {
		return err
	}

	return nil
}
