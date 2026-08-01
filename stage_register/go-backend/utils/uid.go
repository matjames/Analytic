package utils

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"
)

const (
	LETTERS    = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	ALPHANUM   = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	UID_LENGTH = 11
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func GenerateUid() string {
	uid := ""
	// first char must be a letter
	uid += string(LETTERS[rng.Intn(len(LETTERS))])
	for i := 1; i < UID_LENGTH; i++ {
		uid += string(ALPHANUM[rng.Intn(len(ALPHANUM))])
	}
	return uid
}

// EnsureUniqueMflUid generates a unique UID for a given table
func EnsureUniqueMflUid(db *sql.DB, tableName string) (string, error) {
	attempts := 0
	for attempts < 100 {
		candidate := GenerateUid()
		var exists int
		err := db.QueryRow(
			"SELECT 1 FROM "+tableName+" WHERE mfl_uid = $1 LIMIT 1",
			candidate,
		).Scan(&exists)

		if err == sql.ErrNoRows {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		attempts++
	}
	return "", sql.ErrNoRows // Return error if we couldn't generate unique UID
}

// MflUidExistsInFacilityOrAdminUnit returns true if the given mfl_uid is already used in facilities or admin_units
func MflUidExistsInFacilityOrAdminUnit(db *sql.DB, mflUid string) (bool, error) {
	var exists int
	err := db.QueryRow(
		`SELECT 1 FROM (
			SELECT mfl_uid FROM facilities WHERE mfl_uid = $1
			UNION ALL
			SELECT mfl_uid FROM admin_units WHERE mfl_uid = $1
		) u LIMIT 1`,
		mflUid,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// EnsureUniqueMflUidForFacility generates a UID that does not exist in either facilities or admin_units
func EnsureUniqueMflUidForFacility(db *sql.DB) (string, error) {
	for attempts := 0; attempts < 100; attempts++ {
		candidate := GenerateUid()
		exists, err := MflUidExistsInFacilityOrAdminUnit(db, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not generate unique mfl_uid")
}

// ResolveUniqueMflUid returns the candidate if it is not yet used, otherwise generates a new unique mfl_uid
func ResolveUniqueMflUid(db *sql.DB, candidate string) (string, error) {
	if candidate == "" {
		return EnsureUniqueMflUidForFacility(db)
	}
	exists, err := MflUidExistsInFacilityOrAdminUnit(db, candidate)
	if err != nil {
		return "", err
	}
	if !exists {
		return candidate, nil
	}
	return EnsureUniqueMflUidForFacility(db)
}

// GenerateFacilityIdentifier generates a facility identifier
func GenerateFacilityIdentifier() string {
	const PREFIX = "800802"
	const MAX_SUFFIX = 9999999 // 7-digit upper bound

	// Generate random number between 0 and MAX_SUFFIX
	suffix := rng.Intn(MAX_SUFFIX + 1)
	// Pad to 7 digits
	suffixStr := fmt.Sprintf("%07d", suffix)
	return PREFIX + suffixStr
}
