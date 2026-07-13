package api

import (
	"context"
	"strings"

	"github.com/jmoiron/sqlx"
)

type MasterDataProvenance struct {
	CreatedAt string `json:"createdAt" db:"created_at"`
	CreatedBy *int   `json:"createdBy" db:"created_by"`
	ChangedAt string `json:"changedAt" db:"changed_at"`
	ChangedBy *int   `json:"changedBy" db:"changed_by"`
}

func provenanceActor(ctx context.Context) any {
	if userID, err := currentUserID(ctx); err == nil {
		return userID
	}
	return nil
}

func setCreatedProvenance(ctx context.Context, db sqlx.ExtContext, table string, id int) error {
	return setMasterDataProvenance(ctx, db, table, id, true)
}

func setChangedProvenance(ctx context.Context, db sqlx.ExtContext, table string, id int) error {
	return setMasterDataProvenance(ctx, db, table, id, false)
}

func setMasterDataProvenance(ctx context.Context, db sqlx.ExtContext, table string, id int, created bool) error {
	var err error
	if created {
		_, err = db.ExecContext(ctx, "UPDATE "+table+" SET created_by = ?, changed_by = ? WHERE id = ?", provenanceActor(ctx), provenanceActor(ctx), id)
	} else {
		_, err = db.ExecContext(ctx, "UPDATE "+table+" SET changed_by = ? WHERE id = ?", provenanceActor(ctx), id)
	}
	// Small unit-test schemas and pre-migration development databases may not
	// expose provenance columns. Production startup always runs migrations.
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "no such column") {
		return nil
	}
	return err
}
