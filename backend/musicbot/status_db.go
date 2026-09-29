package musicbot

import (
	"context"
	"errors"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/jackc/pgx/v5"
)

const (
	IncidentAuto        = "auto"
	IncidentNotice      = "notice"
	IncidentMaintenance = "maintenance"
)

var ErrIncidentNotFound = errors.New("incident not found")

type StatusSample struct {
	Component string
	Start     time.Time
	Up        bool
}

type StatusBucket struct {
	Component string
	Start     time.Time
	Up        int
	Total     int
}

type Incident struct {
	ID            int64        `json:"id"`
	Component     string       `json:"component,omitempty"`
	ComponentName string       `json:"component_name,omitempty"`
	Kind          string       `json:"kind"`
	Title         string       `json:"title"`
	Message       string       `json:"message,omitempty"`
	StartedAt     time.Time    `json:"started_at"`
	ResolvedAt    *time.Time   `json:"resolved_at,omitempty"`
	CreatedBy     snowflake.ID `json:"created_by,omitempty"`
}

func (d *DB) AddStatusSamples(ctx context.Context, samples []StatusSample) error {
	if len(samples) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, s := range samples {
		up := 0
		if s.Up {
			up = 1
		}
		batch.Queue(`
			INSERT INTO status_samples (component, bucket_start, up, total) VALUES ($1, $2, $3, 1)
			ON CONFLICT (component, bucket_start) DO UPDATE
			SET up = status_samples.up + EXCLUDED.up, total = status_samples.total + 1`,
			s.Component, s.Start, up)
	}
	return d.Pool.SendBatch(ctx, batch).Close()
}

func scanBuckets(rows pgx.Rows) ([]StatusBucket, error) {
	defer rows.Close()
	out := make([]StatusBucket, 0)
	for rows.Next() {
		var b StatusBucket
		if err := rows.Scan(&b.Component, &b.Start, &b.Up, &b.Total); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (d *DB) StatusBuckets(ctx context.Context, since time.Time) ([]StatusBucket, error) {
	rows, err := d.Pool.Query(ctx,
		"SELECT component, bucket_start, up, total FROM status_samples WHERE bucket_start >= $1 ORDER BY bucket_start", since)
	if err != nil {
		return nil, err
	}
	return scanBuckets(rows)
}

func (d *DB) StatusDaily(ctx context.Context, since time.Time) ([]StatusBucket, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT component, date_trunc('day', bucket_start AT TIME ZONE 'UTC') AT TIME ZONE 'UTC', sum(up)::int, sum(total)::int
		FROM status_samples WHERE bucket_start >= $1
		GROUP BY 1, 2 ORDER BY 2`, since)
	if err != nil {
		return nil, err
	}
	return scanBuckets(rows)
}

func (d *DB) FirstStatusSample(ctx context.Context) (time.Time, error) {
	var first *time.Time
	if err := d.Pool.QueryRow(ctx, "SELECT min(bucket_start) FROM status_samples").Scan(&first); err != nil || first == nil {
		return time.Time{}, err
	}
	return *first, nil
}

func (d *DB) PruneStatus(ctx context.Context, before time.Time) error {
	_, err := d.Pool.Exec(ctx, "DELETE FROM status_samples WHERE bucket_start < $1", before)
	return err
}

const incidentColumns = "id, component, component_name, kind, title, message, started_at, resolved_at, COALESCE(created_by, 0)"

func scanIncidents(rows pgx.Rows) ([]Incident, error) {
	defer rows.Close()
	out := make([]Incident, 0)
	for rows.Next() {
		var (
			in        Incident
			createdBy int64
		)
		if err := rows.Scan(&in.ID, &in.Component, &in.ComponentName, &in.Kind, &in.Title, &in.Message,
			&in.StartedAt, &in.ResolvedAt, &createdBy); err != nil {
			return nil, err
		}
		in.CreatedBy = snowflake.ID(createdBy)
		out = append(out, in)
	}
	return out, rows.Err()
}

func (d *DB) CreateIncident(ctx context.Context, in *Incident) error {
	var createdBy *int64
	if in.CreatedBy != 0 {
		id := int64(in.CreatedBy)
		createdBy = &id
	}
	if in.StartedAt.IsZero() {
		in.StartedAt = time.Now()
	}
	return d.Pool.QueryRow(ctx, `
		INSERT INTO status_incidents (component, component_name, kind, title, message, started_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		in.Component, in.ComponentName, in.Kind, Trim(in.Title, 255), in.Message, in.StartedAt, createdBy,
	).Scan(&in.ID)
}

func (d *DB) ResolveIncident(ctx context.Context, id int64, at time.Time) error {
	tag, err := d.Pool.Exec(ctx, "UPDATE status_incidents SET resolved_at = $1 WHERE id = $2 AND resolved_at IS NULL", at, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrIncidentNotFound
	}
	return nil
}

func (d *DB) DeleteIncident(ctx context.Context, id int64) error {
	tag, err := d.Pool.Exec(ctx, "DELETE FROM status_incidents WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrIncidentNotFound
	}
	return nil
}

func (d *DB) OpenAutoIncidents(ctx context.Context) ([]Incident, error) {
	rows, err := d.Pool.Query(ctx, "SELECT "+incidentColumns+" FROM status_incidents WHERE kind = $1 AND resolved_at IS NULL", IncidentAuto)
	if err != nil {
		return nil, err
	}
	return scanIncidents(rows)
}

func (d *DB) StatusIncidents(ctx context.Context, since time.Time, limit int) ([]Incident, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT `+incidentColumns+` FROM status_incidents
		WHERE started_at >= $1 OR resolved_at IS NULL
		ORDER BY started_at DESC LIMIT $2`, since, limit)
	if err != nil {
		return nil, err
	}
	return scanIncidents(rows)
}
