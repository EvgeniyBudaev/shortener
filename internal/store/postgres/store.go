package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"github.com/EvgeniyBudaev/shortener/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"runtime"
)

// DBStore - Интерфейс работы с пулом соединений.
type DBStore struct {
	conn *pgxpool.Pool
}

// ErrDBInsertConflict Обнаружен конфликт в БД, необходимо его обработать.
var ErrDBInsertConflict = errors.New("conflict insert into table, returned stored value")

// ErrURLDeleted Запрашиваемый URL удален.
var ErrURLDeleted = errors.New("url is deleted")

// NewPostgresStore Функция получения экземпляра DBStore.
func NewPostgresStore(ctx context.Context, dsn string) (*DBStore, error) {
	if err := runMigrations(dsn); err != nil {
		return nil, fmt.Errorf("failed to run DB migrations: %w", err)
	}
	conf, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	conf.MaxConns = int32(runtime.NumCPU() * 4)
	conn, err := pgxpool.NewWithConfig(ctx, conf)
	if err != nil {
		return nil, err
	}
	dbStore := &DBStore{conn: conn}
	return dbStore, nil
}

//go:embed migrations/*.sql
var migrationsDir embed.FS

// runMigrations Применение миграций из папки в текущем каталоге - migrations.
func runMigrations(dsn string) error {
	d, err := iofs.New(migrationsDir, "migrations")
	if err != nil {
		return fmt.Errorf("failed to return an iofs driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		return fmt.Errorf("failed to get a new migrate instance: %w", err)
	}
	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("failed to apply migrations to the DB: %w", err)
		}
	}
	return nil
}

func (db *DBStore) Ping() error {
	return db.conn.Ping(context.Background())
}

func (db *DBStore) Close() {
	db.conn.Close()
}

func (db *DBStore) Get(ctx *gin.Context, id string) (string, error) {
	row := db.conn.QueryRow(ctx,
		"SELECT original_url, deleted_flag FROM shortener WHERE slug = $1", id)
	var result string
	var deleted bool
	err := row.Scan(&result, &deleted)
	if err != nil {
		return "", err
	}
	if deleted {
		return "", ErrURLDeleted
	}
	return result, nil
}

func (db *DBStore) GetAllByUserID(ctx *gin.Context, userID string) ([]models.URLRecord, error) {
	result := make([]models.URLRecord, 0)

	rows, err := db.conn.Query(ctx, `
		SELECT slug, original_url
		FROM shortener
		WHERE user_id = $1 AND deleted_flag = FALSE
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		record := models.URLRecord{}
		if err := rows.Scan(&record.ShortURL, &record.OriginalURL); err != nil {
			return nil, err
		}

		result = append(result, record)
	}

	return result, nil
}

func (db *DBStore) DeleteMany(ctx *gin.Context, ids models.DeleteUserURLsReq, userID string) error {
	query := `
		UPDATE shortener SET deleted_flag = TRUE
		WHERE shortener.slug = $1 AND shortener.user_id = $2`
	batch := &pgx.Batch{}
	for _, url := range ids {
		batch.Queue(query, url, userID)
	}
	batchResults := db.conn.SendBatch(ctx, batch)
	defer batchResults.Close()

	for range ids {
		_, err := batchResults.Exec()
		if err != nil {
			log.Printf("error executing: %v", err)
			return err
		}
	}

	return nil
}

func (db *DBStore) Put(ctx *gin.Context, id string, url string, userID string) (string, error) {
	var err error

	row := db.conn.QueryRow(ctx, `
		INSERT INTO shortener VALUES ($1, $2, $3)
		ON CONFLICT (original_url)
		DO UPDATE SET
			original_url=EXCLUDED.original_url
		RETURNING slug
	`, id, url, userID)
	var result string
	if err := row.Scan(&result); err != nil {
		return "", err
	}

	if id != result {
		err = ErrDBInsertConflict
	}

	return result, err
}

func (db *DBStore) PutBatch(ctx *gin.Context, urls []models.URLBatchReq, userID string) ([]models.URLBatchRes, error) {
	query := `
		INSERT INTO shortener VALUES (@slug, @originalUrl, @userID)
		ON CONFLICT (original_url)
		DO UPDATE SET
			original_url=EXCLUDED.original_url
		RETURNING slug	
	`
	result := make([]models.URLBatchRes, 0)

	batch := &pgx.Batch{}
	for _, url := range urls {
		args := pgx.NamedArgs{
			"slug":        url.CorrelationID,
			"originalUrl": url.OriginalURL,
			"userID":      userID,
		}
		batch.Queue(query, args)
	}
	results := db.conn.SendBatch(ctx, batch)
	defer results.Close()

	for _, url := range urls {
		id, err := results.Exec()
		if err != nil {
			return nil, err
		}
		result = append(result, models.URLBatchRes{
			CorrelationID: url.CorrelationID,
			ShortURL:      id.String(),
		})
	}

	return result, nil
}
