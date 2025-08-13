package repository
import(
	"github.com/jackc/pgx/v5/pgxpool"
	"context"
	"fmt"
)

type PostgresRepo struct {
	db *pgxpool.Pool
}

func NewPostgresRepo(connString string) (*PostgresRepo, error) {
	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("DB connection failed: %w", err)
	}
	return &PostgresRepo{db: pool}, nil
}