package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"person-service/internal/model"

	"github.com/lib/pq"
	"go.uber.org/zap"
)

type PostgresRepository struct {
	db     *sql.DB
	logger *zap.Logger
	config *DBConfig
}

type DBConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func NewPostgresRepository(connString string, cfg *DBConfig, logger *zap.Logger) (*PostgresRepository, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	// logger.Info("database connection pool configured",
	// 	zap.Int("max_open_conns", cfg.MaxOpenConns),
	// 	zap.Int("max_idle_conns", cfg.MaxIdleConns),
	// 	zap.Duration("conn_max_lifetime", cfg.ConnMaxLifetime),
	// 	zap.Duration("conn_max_idle_time", cfg.ConnMaxIdleTime),
	// )

	go monitorDBPool(db, logger)

	return &PostgresRepository{
		db:     db,
		logger: logger,
		config: cfg,
	}, nil
}

func monitorDBPool(db *sql.DB, logger *zap.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := db.Stats()
		logger.Debug("database pool stats",
			zap.Int("max_open_conns", stats.MaxOpenConnections),
			zap.Int("open_conns", stats.OpenConnections),
			zap.Int("in_use", stats.InUse),
			zap.Int("idle", stats.Idle),
			zap.Int64("wait_count", stats.WaitCount),
			zap.Duration("wait_duration", stats.WaitDuration),
			zap.Int64("max_idle_closed", stats.MaxIdleClosed),
			zap.Int64("max_lifetime_closed", stats.MaxLifetimeClosed),
		)

		// Предупреждение, если слишком много ожидающих соединений
		if stats.WaitCount > 100 {
			logger.Warn("high database connection wait count",
				zap.Int64("wait_count", stats.WaitCount),
				zap.Duration("wait_duration", stats.WaitDuration),
			)
		}

		// Предупреждение, если все соединения заняты
		if stats.InUse == stats.MaxOpenConnections && stats.MaxOpenConnections > 0 {
			logger.Warn("all database connections are in use",
				zap.Int("in_use", stats.InUse),
				zap.Int("max_open", stats.MaxOpenConnections),
			)
		}
	}
}

func (r *PostgresRepository) Create(ctx context.Context, person *model.Person) error {
	query := `
        INSERT INTO people (first_name, last_name, patronymic, age, gender, nationality, emails)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := r.db.QueryRowContext(ctx, query,
		person.FirstName, person.LastName, person.Patronymic,
		person.Age, person.Gender, person.Nationality,
		pq.Array(person.Emails),
	).Scan(&person.ID)

	if err != nil {
		return fmt.Errorf("query failed for person %s %s: %w", person.FirstName, person.LastName, err)
	}

	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id int64) (*model.Person, error) {
	query := `
        SELECT id, first_name, last_name, patronymic, age, gender, nationality, emails, created_at, updated_at
        FROM people WHERE id = $1
    `
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var person model.Person
	var emails []string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&person.ID, &person.FirstName, &person.LastName, &person.Patronymic,
		&person.Age, &person.Gender, &person.Nationality,
		pq.Array(&emails), &person.CreatedAt, &person.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("person with id %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("query failed %d: %w", id, err)
	}

	person.Emails = emails
	return &person, nil
}

func (r *PostgresRepository) List(ctx context.Context, page, pageSize int32) ([]*model.Person, int32, error) {
	offset := (page - 1) * pageSize

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var totalCount int32
	countQuery := "SELECT COUNT(*) FROM people"
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to get count: %w", err)
	}

	query := `
        SELECT id, first_name, last_name, patronymic, age, gender, nationality, emails, created_at, updated_at
        FROM people
        ORDER BY id
        LIMIT $1 OFFSET $2
    `

	rows, err := r.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var persons []*model.Person
	for rows.Next() {
		var person model.Person
		var emails []string

		err := rows.Scan(
			&person.ID, &person.FirstName, &person.LastName, &person.Patronymic,
			&person.Age, &person.Gender, &person.Nationality,
			pq.Array(&emails), &person.CreatedAt, &person.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan row: %w", err)
		}

		person.Emails = emails
		persons = append(persons, &person)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	return persons, totalCount, nil
}

func (r *PostgresRepository) Update(ctx context.Context, existing *model.Person, person *model.UpdatePersonDTO) error {
	updColumns := []string{}
	args := []interface{}{}
	argCounter := 1

	if person.FirstName != nil && *person.FirstName != existing.FirstName {
		updColumns = append(updColumns, fmt.Sprintf("first_name = $%d", argCounter))
		args = append(args, *person.FirstName)
		argCounter++
	}

	if person.LastName != nil && *person.LastName != existing.LastName {
		updColumns = append(updColumns, fmt.Sprintf("last_name = $%d", argCounter))
		args = append(args, *person.LastName)
		argCounter++
	}

	if person.Patronymic != nil && *person.Patronymic != *existing.Patronymic {
		updColumns = append(updColumns, fmt.Sprintf("patronymic = $%d", argCounter))
		args = append(args, *person.Patronymic)
		argCounter++
	}

	if person.Age != nil && *person.Age != int32(existing.Age) {
		updColumns = append(updColumns, fmt.Sprintf("age = $%d", argCounter))
		args = append(args, *person.Age)
		argCounter++
	}

	if person.Gender != nil && *person.Gender != existing.Gender {
		updColumns = append(updColumns, fmt.Sprintf("gender = $%d", argCounter))
		args = append(args, *person.Gender)
		argCounter++
	}

	if person.Nationality != nil && *person.Nationality != existing.Nationality {
		updColumns = append(updColumns, fmt.Sprintf("nationality = $%d", argCounter))
		args = append(args, *person.Nationality)
		argCounter++
	}

	if person.Emails != nil {
		var upd bool
		if len(person.Emails) == len(existing.Emails) {
			for i, e := range person.Emails {
				if e != existing.Emails[i] {
					upd = true
					break
				}
			}
		} else {
			upd = true
		}

		if upd {
			updColumns = append(updColumns, fmt.Sprintf("emails = $%d", argCounter))
			args = append(args, person.Emails)
			argCounter++
		}
	}

	if len(updColumns) == 0 {
		return fmt.Errorf("no fields to update for person %d", person.ID)
	}

	query := fmt.Sprintf("UPDATE people SET %s WHERE id = $%d",
		strings.Join(updColumns, ", "), argCounter,
	)
	args = append(args, person.ID)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := r.db.ExecContext(ctx, query, args...)

	if err != nil {
		return fmt.Errorf("failed to update person %d: %w", person.ID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for person %d: %w", person.ID, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("person with id %d not found", person.ID)
	}

	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id int64) error {
	query := "DELETE FROM people WHERE id = $1"

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete person %d: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected %d: %w", id, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("person with id %d not found", id)
	}

	return nil
}

func (r *PostgresRepository) Close() error {
	r.logger.Info("closing database connection pool")

	return r.db.Close()
}
