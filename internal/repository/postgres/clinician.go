package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type clinicianRepository struct {
	*BaseRepository
}

func NewClinicianRepository(base BaseRepository) repository.ClinicianRepository {
	return &clinicianRepository{
		BaseRepository: &base,
	}
}

func (r *clinicianRepository) Get(ctx context.Context, id uuid.UUID) (*model.Clinician, error) {
	query := `SELECT * FROM clinicians WHERE id = $1`
	var clinician model.Clinician
	err := r.GetDB().GetContext(ctx, &clinician, query, id)
	return &clinician, err
}
