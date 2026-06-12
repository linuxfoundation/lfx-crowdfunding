// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package snowflake

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// fixtureProgram mirrors MentorshipProgram with JSON tags for fixture file decoding.
type fixtureProgram struct {
	JobspringProjectID string               `json:"jobspring_project_id"`
	Name               string               `json:"name"`
	Status             string               `json:"status"`
	Beneficiaries      []fixtureBeneficiary `json:"beneficiaries"`
}

type fixtureBeneficiary struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// FixtureSource implements mentorshipSource by reading a JSON file from disk.
// Used in DEV and local development — requires no Snowflake credentials.
type FixtureSource struct {
	path string
}

// NewFixtureSource returns a FixtureSource that reads programs from the given path.
func NewFixtureSource(path string) *FixtureSource {
	return &FixtureSource{path: path}
}

// FetchPrograms reads the fixture file and returns its contents as domain models.
func (f *FixtureSource) FetchPrograms(_ context.Context) ([]models.MentorshipProgram, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		return nil, fmt.Errorf("read fixture file %q: %w", f.path, err)
	}

	var raw []fixtureProgram
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse fixture file %q: %w", f.path, err)
	}

	programs := make([]models.MentorshipProgram, len(raw))
	for i, r := range raw {
		beneficiaries := make([]models.MentorshipBeneficiary, len(r.Beneficiaries))
		for j, b := range r.Beneficiaries {
			beneficiaries[j] = models.MentorshipBeneficiary{Name: b.Name, Email: b.Email}
		}
		programs[i] = models.MentorshipProgram{
			JobspringProjectID: r.JobspringProjectID,
			Name:               r.Name,
			Status:             r.Status,
			Beneficiaries:      beneficiaries,
		}
	}
	return programs, nil
}
