package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dezzerlol/avitotech-test-2023/internal/db/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Segment struct {
	DB *pgxpool.Pool
}

func NewSegment(db *pgxpool.Pool) *Segment {
	return &Segment{DB: db}
}

var (
	ErrSegmentNotFound = errors.New("segment not found")
)

func (r Segment) Create(ctx context.Context, segment *models.Segment) error {
	query := `
		INSERT INTO segments (slug)
		VALUES ($1)
		RETURNING id, created_at
	`

	args := []any{
		segment.Slug,
	}

	err := r.DB.
		QueryRow(ctx, query, args...).
		Scan(&segment.Id, &segment.CreatedAt)

	return err
}

func (r Segment) DeleteBySlug(ctx context.Context, segment *models.Segment) error {
	query := `
		DELETE FROM segments
		WHERE slug = $1
	`

	args := []any{&segment.Slug}

	ct, err := r.DB.Exec(ctx, query, args...)

	if ct.RowsAffected() == 0 {
		return ErrSegmentNotFound
	}

	return err
}

func (r Segment) GetUserSegments(ctx context.Context, userId int64) ([]*models.Segment, error) {
	query := `
		SELECT slug
		FROM segments s
		JOIN user_segments us
		on s.id = us.segment_id
		WHERE us.user_id = $1
	`

	args := []any{userId}

	rows, err := r.DB.Query(ctx, query, args...)

	if err != nil {
		return nil, err
	}

	var segments []*models.Segment

	for rows.Next() {
		var segment models.Segment

		err := rows.Scan(
			&segment.Slug,
		)

		if err != nil {
			return nil, err
		}

		segments = append(segments, &segment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return segments, nil
}

func (r Segment) AddUserSegments(ctx context.Context, userId int64, addSegments []string) (int64, error) {
	var sb strings.Builder

	// Сначала получаем id сегментов, которые нужно добавить
	// Потом вставляем в user_segments
	sb.WriteString(`
	INSERT INTO user_segments (segment_id, user_id)
	SELECT s.id, $1
	FROM segments s
	WHERE s.slug IN (
	`)

	args := []any{userId}

	// Готовим аргументы для запроса
	// Добавялем 2 потому что первый аргумент это userId
	for i, slug := range addSegments {
		args = append(args, slug)
		sb.WriteString(fmt.Sprintf("$%d", i+2))

		if i != len(addSegments)-1 {
			sb.WriteString(",")
		}
	}

	// Закрываем строку запроса
	// Добавляем пропуск конфликта, если сегмент уже есть у пользователя
	sb.WriteString(") ON CONFLICT DO NOTHING")

	query := sb.String()

	ct, err := r.DB.Exec(ctx, query, args...)

	return ct.RowsAffected(), err
}

func (r Segment) DeleteUserSegments(ctx context.Context, userId int64, deleteSegments []string) (int64, error) {
	var sb strings.Builder

	// Сначала получаем id сегментов, которые нужно удалить
	// Потом удаляем из user_segments
	sb.WriteString(`
	DELETE FROM user_segments us
	WHERE us.user_id = $1
	AND us.segment_id IN (
		SELECT s.id
		FROM segments s
		WHERE s.slug IN (
	`)

	args := []any{userId}

	// Готовим аргументы для запроса
	// Добавялем 2 потому что первый аргумент это userId
	for i, slug := range deleteSegments {
		args = append(args, slug)
		sb.WriteString(fmt.Sprintf("$%d", i+2))

		if i != len(deleteSegments)-1 {
			sb.WriteString(",")
		}
	}

	// Закрываем строку запроса
	sb.WriteString("))")

	query := sb.String()

	ct, err := r.DB.Exec(ctx, query, args...)

	return ct.RowsAffected(), err
}