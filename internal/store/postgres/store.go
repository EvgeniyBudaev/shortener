package postgres

import (
	"context"
	"errors"
	"github.com/EvgeniyBudaev/shortener/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBStore struct {
	conn *pgxpool.Pool
}

var ErrDBInsertConflict = errors.New("conflict insert into table, returned stored value")

func NewPostgresStore(dsn string) (*DBStore, error) {
	conn, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	dbStore := &DBStore{conn: conn}

	if err := dbStore.CreateTable(); err != nil {
		return nil, err
	}

	return dbStore, nil
}

func (db *DBStore) Ping() error {
	return db.conn.Ping(context.Background())
}

func (db *DBStore) Get(ctx *gin.Context, id string) (string, error) {
	row := db.conn.QueryRow(ctx, "SELECT original_url FROM shortener WHERE slug = $1", id)
	var result string
	err := row.Scan(&result)
	if err != nil {
		return "", err
	}
	return result, nil
}

func (db *DBStore) Put(ctx *gin.Context, id string, url string) (string, error) {
	var err error

	row := db.conn.QueryRow(ctx, `
		INSERT INTO shortener VALUES ($1, $2)
		ON CONFLICT (original_url)
		DO UPDATE SET
			original_url=EXCLUDED.original_url
		RETURNING slug
	`, id, url)
	var result string
	if err := row.Scan(&result); err != nil {
		return "", err
	}

	if id != result {
		err = ErrDBInsertConflict
	}

	return result, err
}

func (db *DBStore) PutBatch(ctx *gin.Context, urls []models.URLBatchReq) ([]models.URLBatchRes, error) {
	query := `
		INSERT INTO shortener VALUES (@slug, @originalUrl)
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

func (db *DBStore) CreateTable() error {
	_, err := db.conn.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS shortener(
		slug VARCHAR(255),
		original_url VARCHAR(255) PRIMARY KEY,
		UNIQUE(slug, original_url)
	);`)
	return err
}
