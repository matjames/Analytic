// Command seed_users creates 200 realistic StatGate Analytics staff accounts in
// the Registry identity database (kaggle) and migrates the full Registry user
// roster into the StatChat database so both services share the same people.
//
// Accounts use e-mail addresses on @statgate.analytics.gov, usernames in the
// form surname.firstinitial, and a common bcrypt-hashed onboarding password.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const (
	targetUsers     = 200
	defaultPassword = "Statgate@2026" // meets policy: >=10 chars, upper, lower, digit
	orgID           = "statgate.analytics.gov"
	registryDSNEnv  = "SEED_REGISTRY_DSN"
	statchatDSNEnv  = "SEED_STATCHAT_DSN"
	registryDSNDef  = "host=localhost port=5432 user=Kaggle password=Statgate_kaggle dbname=kaggle sslmode=disable"
	statchatDSNDef  = "host=localhost port=5432 user=Statchat password=Statgate dbname=statchat sslmode=disable"
	generalConvID   = "general"
)

type seedUser struct {
	FirstName string
	LastName  string
	Username  string
	Email     string
	Role      string
	District  string
	Phone     string
}

// 100 first names and 100 surnames give exactly 200 unique full-name pairings
// (second block re-pairs surnames with a coprime stride).
var firstNames = []string{
	"John", "Grace", "Peter", "Sarah", "David", "Ruth", "Joseph", "Esther", "Samuel", "Joy",
	"Moses", "Agnes", "Daniel", "Faith", "Isaac", "Miriam", "James", "Peace", "Andrew", "Rose",
	"Robert", "Charity", "Francis", "Beatrice", "Ronald", "Juliet", "Patrick", "Alice", "Edward", "Diana",
	"Charles", "Gloria", "Vincent", "Rebecca", "Simon", "Phionah", "Henry", "Joan", "Godfrey", "Maria",
	"Martin", "Harriet", "Denis", "Joyce", "Emmanuel", "Lillian", "Geoffrey", "Sylvia", "Paul", "Florence",
	"Fred", "Claire", "Michael", "Sophia", "Chris", "Evelyn", "George", "Winnie", "Julius", "Anthony",
	"Bridget", "Richard", "Immaculate", "Kenneth", "Mary", "Brian", "Noeline", "Alex", "Ivan", "Susan",
	"Derrick", "Nicholas", "Timothy", "Wilberforce", "Catherine", "Monica", "Eva", "Benon", "Judith", "Olivia",
	"Lawrence", "Priscilla", "Annet", "Dan", "Robinah", "Teopista", "Mark", "Teddy", "Rogers", "Margret",
	"Aidah", "Siraje", "Moreen", "Hassan", "Hadijah", "Musa", "Nusurah", "Ibrahim", "Amina", "Yusuf",
}

var lastNames = []string{
	"Matovu", "Kato", "Namusoke", "Okello", "Mugisha", "Nakato", "Ssebunya", "Achieng", "Tumusiime", "Nansubuga",
	"Ssemakula", "Ochieng", "Babirye", "Wasswa", "Nakabugo", "Lule", "Kiyingi", "Nabirye", "Odeke", "Auma",
	"Kiggundu", "Nabatanzi", "Byaruhanga", "Nassali", "Mugerwa", "Nakyanzi", "Ssentongo", "Nakimuli", "Ochen", "Adong",
	"Kizza", "Nantongo", "Tumwine", "Nakazibwe", "Oloya", "Aol", "Mukasa", "Namaganda", "Ssekandi", "Nakitto",
	"Okwir", "Kasirye", "Wandera", "Amongin", "Musoke", "Namuli", "Ojok", "Ayikoru", "Ssebina", "Nakalema",
	"Mpagi", "Nakigudde", "Odongo", "Abbo", "Baluku", "Nakabale", "Katongole", "Namazzi", "Kirunda", "Nabukeera",
	"Oboth", "Adiru", "Ssempijja", "Nabawanuka", "Onyango", "Achola", "Lukwago", "Nakaweesi", "Okurut", "Amongi",
	"Ssentamu", "Nansamba", "Karuhanga", "Opiyo", "Apio", "Ssekitoleko", "Nabukenya", "Okoth", "Aber", "Katusiime",
	"Namukasa", "Etori", "Nambooze", "Bwanika", "Nakachwa", "Seekayiba", "Nakiganda", "Oyet", "Aciro", "Ssemwogerere",
	"Nabagala", "Kintu", "Okia", "Amuge", "Kalanzi", "Turyahebwa", "Nassimbwa", "Mukiibi", "Naluyima", "Ocen",
}
var districts = []string{
	"Kampala", "Wakiso", "Mukono", "Jinja", "Mbarara", "Gulu", "Lira", "Mbale", "Soroti", "Arua",
	"Fort Portal", "Masaka", "Entebbe", "Hoima", "Kabale", "Tororo", "Busia", "Iganga", "Kayunga", "Luwero",
	"Kalangala", "Kiboga", "Rakai", "Sembabule", "Kyenjojo",
}

var baseRoles = []string{
	"analyst", "operator", "viewer", "manager", "editor", "analyst", "analyst", "operator",
	"viewer", "analyst", "editor", "operator", "manager", "analyst", "viewer", "analyst",
}

// preferredRole overrides a realistic portion of the roster with senior roles.
func preferredRole(i int, base string) string {
	switch {
	case i%33 == 0:
		return "admin"
	case i%21 == 0:
		return "district_admin"
	case i%50 == 0:
		return "governance_officer"
	case i%17 == 0:
		return "tenant_admin"
	}
	return base
}

func buildUsers() ([]seedUser, error) {
	users := make([]seedUser, 0, targetUsers)
	seenEmail := map[string]bool{}
	usernameCount := map[string]int{}
	for i := 0; i < targetUsers; i++ {
		fi := i % len(firstNames)
		li := (i%len(lastNames) + 37*(i/len(firstNames))) % len(lastNames)
		fn := firstNames[fi]
		ln := lastNames[li]
		email := strings.ToLower(fn+"."+ln) + "@statgate.analytics.gov"
		if seenEmail[email] {
			return nil, fmt.Errorf("duplicate email %s at index %d", email, i)
		}
		seenEmail[email] = true

		baseUsername := strings.ToLower(ln) + "." + strings.ToLower(string(fn[0]))
		count := usernameCount[baseUsername]
		usernameCount[baseUsername] = count + 1
		username := baseUsername
		if count > 0 {
			username = fmt.Sprintf("%s%d", baseUsername, count+1)
		}

		role := preferredRole(i, baseRoles[i%len(baseRoles)])
		users = append(users, seedUser{
			FirstName: fn,
			LastName:  ln,
			Username:  username,
			Email:     email,
			Role:      role,
			District:  districts[i%len(districts)],
			Phone:     fmt.Sprintf("+2567%08d", 70_000_000+i),
		})
	}
	return users, nil
}

func dsn(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
func main() {
	regDSN := dsn(registryDSNEnv, registryDSNDef)
	chatDSN := dsn(statchatDSNEnv, statchatDSNDef)

	reg, err := sql.Open("postgres", regDSN)
	if err != nil {
		log.Fatalf("open registry: %v", err)
	}
	defer reg.Close()
	if err := reg.Ping(); err != nil {
		log.Fatalf("ping registry: %v", err)
	}

	chat, err := sql.Open("postgres", chatDSN)
	if err != nil {
		log.Fatalf("open statchat: %v", err)
	}
	defer chat.Close()
	if err := chat.Ping(); err != nil {
		log.Fatalf("ping statchat: %v", err)
	}

	users, err := buildUsers()
	if err != nil {
		log.Fatalf("build users: %v", err)
	}
	fmt.Printf("Generated %d user records\n", len(users))

	hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), 10)
	if err != nil {
		log.Fatalf("bcrypt hash: %v", err)
	}
	passwordHash := string(hash)
	fmt.Printf("Onboarding password: %s  (hash %s...)\n", defaultPassword, passwordHash[:12])

	// ---- 1. Upsert realistic accounts into the Registry (kaggle) ----
	newIDs := []int64{}
	created, updated := 0, 0
	for _, u := range users {
		var existingID int64
		err := reg.QueryRow(`SELECT id FROM users WHERE username = $1`, u.Username).Scan(&existingID)
		switch {
		case err == sql.ErrNoRows:
			var id int64
			err = reg.QueryRow(`
				INSERT INTO users
					(first_name, last_name, username, email, role, password,
					 organisation, phoneno, district_id, must_change_password, email_verified)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, FALSE, TRUE)
				RETURNING id`,
				u.FirstName, u.LastName, u.Username, u.Email, u.Role, passwordHash,
				orgID, u.Phone, u.District,
			).Scan(&id)
			if err != nil {
				log.Fatalf("insert %s: %v", u.Email, err)
			}
			newIDs = append(newIDs, id)
			created++
		case err == nil:
			_, err = reg.Exec(`
				UPDATE users SET
					first_name = $1, last_name = $2, email = $3, role = $4, password = $5,
					organisation = $6, phoneno = $7, district_id = $8,
					must_change_password = FALSE, email_verified = TRUE
				WHERE id = $9`,
				u.FirstName, u.LastName, u.Email, u.Role, passwordHash,
				orgID, u.Phone, u.District, existingID,
			)
			if err != nil {
				log.Fatalf("update %s: %v", u.Email, err)
			}
			newIDs = append(newIDs, existingID)
			updated++
		default:
			log.Fatalf("lookup %s: %v", u.Username, err)
		}
	}
	fmt.Printf("Registry: created=%d updated=%d (total new ids=%d)\n", created, updated, len(newIDs))

	// ---- 2. Migrate the ENTIRE Registry roster into StatChat ----
	rows, err := reg.Query(`
		SELECT id, first_name, last_name, email, role, organisation
		FROM users ORDER BY id`)
	if err != nil {
		log.Fatalf("query registry users: %v", err)
	}

	type regRow struct {
		ID    int64
		First string
		Last  string
		Email string
		Role  string
		Org   sql.NullString
	}
	var all []regRow
	for rows.Next() {
		var rr regRow
		if err := rows.Scan(&rr.ID, &rr.First, &rr.Last, &rr.Email, &rr.Role, &rr.Org); err != nil {
			log.Fatalf("scan registry user: %v", err)
		}
		all = append(all, rr)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		log.Fatalf("iterate registry users: %v", err)
	}

	migrated := 0
	for _, rr := range all {
		name := strings.TrimSpace(rr.First + " " + rr.Last)
		rolesJSON, err := json.Marshal([]string{rr.Role})
		if err != nil {
			log.Fatalf("marshal roles %s: %v", rr.Email, err)
		}
		org := rr.Org.String
		if strings.TrimSpace(org) == "" {
			org = orgID
		}
		_, err = chat.Exec(`
			INSERT INTO users (id, name, email, organization_id, roles, avatar_url, presence, about)
			VALUES ($1, $2, $3, $4, $5, '', 'offline', '')
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				email = EXCLUDED.email,
				organization_id = EXCLUDED.organization_id,
				roles = EXCLUDED.roles`,
			fmt.Sprint(rr.ID), name, rr.Email, org, rolesJSON)
		if err != nil {
			log.Fatalf("upsert statchat user %s: %v", rr.Email, err)
		}
		migrated++
	}
	fmt.Printf("StatChat: migrated %d users from Registry\n", migrated)

	// ---- 3. Add the new staff to the #general conversation ----
	var rawMemberJSON []byte
	err = chat.QueryRow(`SELECT member_ids FROM conversations WHERE id = $1`, generalConvID).Scan(&rawMemberJSON)
	if err != nil {
		log.Fatalf("load general conversation: %v", err)
	}
	var members []string
	if len(rawMemberJSON) > 0 {
		if err := json.Unmarshal(rawMemberJSON, &members); err != nil {
			log.Fatalf("unmarshal general members: %v", err)
		}
	}
	added := 0
	for _, id := range newIDs {
		s := fmt.Sprint(id)
		found := false
		for _, m := range members {
			if m == s {
				found = true
				break
			}
		}
		if !found {
			members = append(members, s)
			added++
		}
	}
	updatedMembers, err := json.Marshal(members)
	if err != nil {
		log.Fatalf("marshal general members: %v", err)
	}
	if _, err := chat.Exec(`UPDATE conversations SET member_ids = $1 WHERE id = $2`, updatedMembers, generalConvID); err != nil {
		log.Fatalf("update general conversation: %v", err)
	}
	fmt.Printf("StatChat: added %d new staff to #general (total members=%d)\n", added, len(members))

	// ---- 4. Summary ----
	fmt.Println("\n=== SEED SUMMARY ===")
	fmt.Printf("Registry users now: %d\n", len(all))
	fmt.Printf("StatChat users now: %d\n", migrated)
	fmt.Println("\nSample accounts (email / username / password):")
	for i, u := range users[:5] {
		fmt.Printf("  %d. %s / %s / %s\n", i+1, u.Email, u.Username, defaultPassword)
	}
}