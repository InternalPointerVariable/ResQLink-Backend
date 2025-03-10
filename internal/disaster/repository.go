package disaster

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Repository interface {
	GetDisasterReports(ctx context.Context) ([]disasterReportResponse, error)
	CreateDisasterReport(ctx context.Context) error
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

func (r *repository) GetDisasterReports(ctx context.Context) ([]disasterReportResponse, error) {
	query := `SELECT * FROM disaster_reports`
	rows, err := r.querier.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	reports, err := pgx.CollectRows(rows, pgx.RowToStructByName[disasterReportResponse])
	if err != nil {
		return nil, err
	}

	return reports, nil
}

func (r *repository) CreateDisasterReport(ctx context.Context) error {
	return nil
}
