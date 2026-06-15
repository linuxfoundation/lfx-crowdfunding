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
	Description        string               `json:"description"`
	Slug               string               `json:"slug"`
	Industry           string               `json:"industry"`
	OwnerLFUsername    string               `json:"owner_lf_username"`
	Skills             []string             `json:"skills"`
	Mentors            []fixtureMentor      `json:"mentors"`
	Beneficiaries      []fixtureBeneficiary `json:"beneficiaries"`
}

type fixtureBeneficiary struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type fixtureMentor struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
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
		// Preserve nil when the JSON field was absent — nil means "source did not
		// provide data" and causes the syncer to skip the corresponding upsert.
		var beneficiaries []models.MentorshipBeneficiary
		if r.Beneficiaries != nil {
			beneficiaries = make([]models.MentorshipBeneficiary, len(r.Beneficiaries))
			for j, b := range r.Beneficiaries {
				beneficiaries[j] = models.MentorshipBeneficiary{Name: b.Name, Email: b.Email}
			}
		}

		var mentors []models.MentorshipMentor
		if r.Mentors != nil {
			mentors = make([]models.MentorshipMentor, len(r.Mentors))
			for j, m := range r.Mentors {
				mentors[j] = models.MentorshipMentor{Name: m.Name, Email: m.Email, AvatarURL: m.AvatarURL}
			}
		}

		programs[i] = models.MentorshipProgram{
			JobspringProjectID: r.JobspringProjectID,
			Name:               r.Name,
			Status:             r.Status,
			Description:        r.Description,
			Slug:               r.Slug,
			Industry:           r.Industry,
			OwnerLFUsername:    r.OwnerLFUsername,
			Skills:             r.Skills,
			Mentors:            mentors,
			Beneficiaries:      beneficiaries,
		}
	}
	return programs, nil
}
