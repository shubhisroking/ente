package authenticator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	model "github.com/ente-io/museum/ente/authenticator"
	"github.com/ente-io/stacktrace"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Create inserts a new  entry
func (r *Repository) Create(ctx context.Context, userID int64, entry model.CreateEntityRequest) (uuid.UUID, error) {
	id := uuid.New()
	err := r.DB.QueryRow(`INSERT into authenticator_entity(
                         id,
                         user_id,
                         encrypted_data,
                         header) VALUES ($1,$2,$3,$4) RETURNING id`,
		id,                  //$1 id
		userID,              // $2 user_id
		entry.EncryptedData, // $3 encrypted_data
		entry.Header).       // $4 header
		Scan(&id)
	if err != nil {
		return id, stacktrace.Propagate(err, "Failed to createTotpEntry")
	}
	return id, nil
}

func (r *Repository) Get(ctx context.Context, userID int64, id uuid.UUID) (model.Entity, error) {
	res := model.Entity{}
	row := r.DB.QueryRowContext(ctx, `SELECT
	id, user_id, encrypted_data, header, is_deleted, created_at, updated_at
	FROM authenticator_entity
	WHERE  id = $1 AND
	user_id = $2`,
		id,     // $1
		userID, // %2     // $3
	)
	err := row.Scan(&res.ID, &res.UserID, &res.EncryptedData, &res.Header, &res.IsDeleted, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return model.Entity{}, stacktrace.Propagate(err, "failed to getTotpEntry")
	}
	return res, nil
}

func (r *Repository) Delete(ctx context.Context, userID int64, id uuid.UUID) (bool, error) {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE authenticator_entity SET is_deleted = true, encrypted_data = NULL, header = NULL where id=$1 and user_id = $2`,
		id, userID)
	if err != nil {
		return false, stacktrace.Propagate(err, fmt.Sprintf("faield to delele totpEntry with id=%s", id))
	}
	return true, nil
}

func (r *Repository) Update(ctx context.Context, userID int64, req model.UpdateEntityRequest) error {
	result, err := r.DB.ExecContext(ctx,
		`UPDATE authenticator_entity SET encrypted_data = $1, header = $2 where id=$3 and user_id = $4 and is_deleted = FALSE`,
		req.EncryptedData, req.Header, req.ID, userID)
	if err != nil {
		return stacktrace.Propagate(err, "")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return stacktrace.Propagate(err, "")
	}
	if affected != 1 {
		return stacktrace.Propagate(errors.New("exactly one row should be updated"), "")
	}
	return nil
}

// GetDiff returns the &{[]ente.TotpEntity} which have been added or
// modified after the given sinceTime
func (r *Repository) GetDiff(ctx context.Context, userID int64, sinceTime int64, limit int16) ([]model.Entity, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT
       id, user_id, encrypted_data, header, is_deleted, created_at, updated_at
	   FROM authenticator_entity
	   WHERE user_id = $1
	   and updated_at > $2
       ORDER BY updated_at
	   LIMIT $3`,
		userID,    // $1
		sinceTime, // %2
		limit,     // $3
	)
	if err != nil {
		return nil, stacktrace.Propagate(err, "GetDiff query failed")
	}
	return convertRowsToToptEntity(rows)
}

func convertRowsToToptEntity(rows *sql.Rows) ([]model.Entity, error) {
	defer func() {
		if err := rows.Close(); err != nil {
			logrus.Error(err)
		}
	}()
	result := make([]model.Entity, 0)
	for rows.Next() {
		entity := model.Entity{}
		err := rows.Scan(
			&entity.ID, &entity.UserID, &entity.EncryptedData, &entity.Header, &entity.IsDeleted,
			&entity.CreatedAt, &entity.UpdatedAt)
		if err != nil {
			return nil, stacktrace.Propagate(err, "Failed to convert convertRowsToToptEntity")
		}
		result = append(result, entity)
	}
	return result, nil
}
