package database

import (
	"context"
	"database/sql"

	"seo-auditor/backend/internal/models"
)

type SeoReportRepository interface {
	Create(ctx context.Context, report *models.SeoReport) error
	ListRecentByUser(ctx context.Context, userID string, limit int) ([]models.SeoReport, error)
}

type SQLSeoReportRepository struct {
	db *sql.DB
}

func NewSQLSeoReportRepository(db *sql.DB) *SQLSeoReportRepository {
	return &SQLSeoReportRepository{db: db}
}

func (repository *SQLSeoReportRepository) Create(ctx context.Context, report *models.SeoReport) error {
	query := `
insert into seo_reports (url, title, summary, score, report, user_id, device_details)
values ($1, $2, $3, $4, $5, $6, $7)
returning id, created_at
`

	return repository.db.QueryRowContext(
		ctx,
		query,
		report.URL,
		report.Title,
		report.Summary,
		report.Score,
		report.Report,
		report.UserID,
		report.DeviceDetails,
	).Scan(&report.ID, &report.CreatedAt)
}

func (repository *SQLSeoReportRepository) ListRecentByUser(ctx context.Context, userID string, limit int) ([]models.SeoReport, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := repository.db.QueryContext(
		ctx,
		`
select id, user_id, url, title, summary, score, report, device_details, created_at
from seo_reports
where user_id = $1
order by created_at desc
limit $2
`,
		userID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := make([]models.SeoReport, 0)
	for rows.Next() {
		var report models.SeoReport
		var rawReport []byte
		if err := rows.Scan(
			&report.ID,
			&report.UserID,
			&report.URL,
			&report.Title,
			&report.Summary,
			&report.Score,
			&rawReport,
			&report.DeviceDetails,
			&report.CreatedAt,
		); err != nil {
			return nil, err
		}

		report.Report = rawReport
		reports = append(reports, report)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reports, nil
}
