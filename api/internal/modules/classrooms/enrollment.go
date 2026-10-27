package classrooms

import (
	"errors"
	"time"
)

// Enrollment controls new joins. It is not an authorization credential and does
// not revoke the membership of a student who already belongs to this class.
type Enrollment struct {
	Open      bool      `firestore:"open" json:"open"`
	ExpiresAt time.Time `firestore:"expires_at" json:"expires_at"`
}

func DefaultEnrollment(now time.Time) *Enrollment {
	return &Enrollment{Open: true, ExpiresAt: now.Add(30 * 24 * time.Hour)}
}

func ValidateEnrollment(value *Enrollment, now time.Time) error {
	if value == nil || !value.Open {
		return nil
	}
	if !value.ExpiresAt.After(now) || value.ExpiresAt.After(now.Add(90*24*time.Hour)) {
		return errors.New("open enrollment requires an expiry in the next 90 days")
	}
	return nil
}

func MayEnroll(class TeacherClassDocument, existingMember bool, now time.Time) bool {
	if class.Status != "active" {
		return false
	}
	// Legacy classes keep their current join behavior until the teacher edits it.
	return existingMember || class.Enrollment == nil || class.Enrollment.Open && class.Enrollment.ExpiresAt.After(now)
}
