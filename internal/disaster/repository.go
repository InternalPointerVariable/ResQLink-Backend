package disaster

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Repository interface {
	GetDisasterReportsByUser(ctx context.Context, userID string) ([]disasterReportResponse, error)
	CreateDisasterReport(ctx context.Context, arg createDisasterReportRequest) error
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
func (r *repository) GetDisasterReportsByUser(
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

func (r *repository) CreateDisasterReport(ctx context.Context, arg createDisasterReportRequest) error {
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
