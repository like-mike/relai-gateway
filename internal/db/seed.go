package db

import (
	"database/sql"
	"log"
)

func SeedTestData(db *sql.DB) error {
	log.Println("🌱 Seeding test data...")

	// Check if any orgs exist
	var orgCount int
	err := db.QueryRow(`SELECT COUNT(*) FROM orgs`).Scan(&orgCount)
	if err != nil {
		return err
	}
	if orgCount > 0 {
		log.Println("Seed skipped — orgs already exist")
		return nil
	}

	// Insert test orgs
	orgIDs := []string{}
	orgNames := []string{"test-org", "alpha-org", "beta-org", "gamma-org", "delta-org"}
	for _, name := range orgNames {
		var orgID string
		err = db.QueryRow(`
			INSERT INTO orgs (name, quota_tokens)
			VALUES ($1, 1000000)
			RETURNING id
		`, name).Scan(&orgID)
		if err != nil {
			return err
		}
		orgIDs = append(orgIDs, orgID)
	}

	// Insert test users
	userIDs := []string{}
	for i, orgID := range orgIDs {
		var userID string
		err = db.QueryRow(`
			INSERT INTO users (org_id, name, email, role)
			VALUES ($1, $2, $3, 'admin')
			RETURNING id
		`, orgID, "User "+orgNames[i], orgNames[i]+"@relai.dev").Scan(&userID)
		if err != nil {
			return err
		}
		userIDs = append(userIDs, userID)
	}

	// Insert test API keys
	for i, orgID := range orgIDs {
		_, err = db.Exec(`
			INSERT INTO api_keys (org_id, user_id, key, max_tokens)
			VALUES ($1, $2, $3, $4)
		`, orgID, userIDs[i], orgNames[i]+"-api-key", 500000+(i*100000))
		if err != nil {
			return err
		}
	}

	log.Println("Seeded multiple orgs, users, and API keys")
	return nil
}
