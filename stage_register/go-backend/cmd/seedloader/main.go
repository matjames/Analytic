package main

import (
	"database/sql"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func loadEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read %s: %v", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && os.Getenv(parts[0]) == "" {
			_ = os.Setenv(parts[0], parts[1])
		}
	}
}

func main() {
	loadStaging := flag.Bool("load-staging", false, "replace only the two CSV staging tables")
	rebuild := flag.Bool("rebuild", false, "rebuild the StatGate field-operations schema and load all seed data")
	flag.Parse()
	loadEnv(".env")
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("connect to PostgreSQL: %v", err)
	}
	if *rebuild {
		schema, err := os.ReadFile("sql_scripts/statgate_field_operations_schema.sql")
		if err != nil {
			log.Fatal(err)
		}
		if _, err := db.Exec(string(schema)); err != nil {
			log.Fatalf("rebuild schema: %v", err)
		}
		*loadStaging = true
	}

	tables := []string{"mflupload", "orgunits_uploads", "facilities", "admin_units", "level", "ownership", "authority", "users"}
	for _, table := range tables {
		var exists bool
		if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)`, table).Scan(&exists); err != nil {
			log.Fatal(err)
		}
		if !exists {
			fmt.Printf("%s: missing\n", table)
			continue
		}
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM public." + table).Scan(&count); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s: %d rows\n", table, count)
	}

	if !*loadStaging {
		return
	}

	if err := loadCSV(db, "mflupload", "mflupload.csv"); err != nil {
		log.Fatal(err)
	}
	if err := loadCSV(db, "orgunits_uploads", "orgunits_uploads.csv"); err != nil {
		log.Fatal(err)
	}
	if *rebuild {
		if err := buildRegistry(db); err != nil {
			log.Fatal(err)
		}
		if err := loadUsers(db, "users_upload.csv"); err != nil {
			log.Fatal(err)
		}
		if err := loadUsers(db, "users_upload_prod.csv"); err != nil {
			log.Fatal(err)
		}
		if err := seedAdministrator(db); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("Staging import complete.")
}

// seedAdministrator guarantees a known local account for initial registry access.
func seedAdministrator(db *sql.DB) error {
	hash, err := bcrypt.GenerateFromPassword([]byte("StatGateAdmin!2026"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO users (first_name,last_name,username,email,role,password,organisation,must_change_password)
		VALUES ('StatGate','Administrator','admin','admin@statgate.local','admin',$1,'StatGate',FALSE)
		ON CONFLICT (email) DO UPDATE SET first_name=EXCLUDED.first_name,last_name=EXCLUDED.last_name,username=EXCLUDED.username,role=EXCLUDED.role,password=EXCLUDED.password,organisation=EXCLUDED.organisation,must_change_password=FALSE,"updatedAt"=NOW()`, string(hash))
	if err == nil {
		fmt.Println("Administrator account: admin@statgate.local")
	}
	return err
}

func loadUsers(db *sql.DB, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\ufeff")))
	header, err := reader.Read()
	if err != nil {
		return err
	}
	index := map[string]int{}
	for i, column := range header {
		index[strings.ToLower(column)] = i
	}
	firstColumn := "first_name"
	lastColumn := "last_name"
	if _, ok := index[firstColumn]; !ok {
		firstColumn, lastColumn = "firstname", "lastname"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("ChangeMe!2026"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		value := func(column string) string { return record[index[column]] }
		_, err = db.Exec(`INSERT INTO users (first_name,last_name,username,email,role,password,district_id,must_change_password)
			VALUES ($1,$2,$3,$4,$5,$6,$7,TRUE)
			ON CONFLICT (email) DO UPDATE SET first_name=EXCLUDED.first_name,last_name=EXCLUDED.last_name,username=EXCLUDED.username,role=EXCLUDED.role,password=EXCLUDED.password,district_id=EXCLUDED.district_id,must_change_password=TRUE,"updatedAt"=NOW()`,
			value(firstColumn), value(lastColumn), value("username"), value("email"), value("role"), string(hash), value("district_id"))
		if err != nil {
			return fmt.Errorf("load %s row %d: %w", filename, count+2, err)
		}
		count++
	}
	fmt.Printf("%s: loaded %d users\n", filename, count)
	return nil
}

func buildRegistry(db *sql.DB) error {
	statements := []string{
		`INSERT INTO admin_level (mfl_uid,name,level_number) VALUES ('sg-root','StatGate Field Operations',1),('sg-region','Region',2),('sg-district','District',3),('sg-municipality','Municipality / DLG',4),('sg-subcounty','Subcounty',5),('sg-station','Field Survey Station',6)`,
		`INSERT INTO "level" (mfl_uid,code,name) VALUES ('tier-community','Community Enumeration Point','Community Enumeration Point'),('tier-subcounty','Subcounty Field Station','Subcounty Field Station'),('tier-district','District Coordination Hub','District Coordination Hub'),('tier-regional','Regional Operations Hub','Regional Operations Hub')`,
		`INSERT INTO ownership (mfl_uid,code,name) VALUES ('contract-public','GOV','Public Sector'),('contract-nonprofit','PNFP','Non-profit Partner'),('contract-private','PFP','Private Partner')`,
		`INSERT INTO authority (mfl_uid,code,name) SELECT DISTINCT 'agency-' || md5(authority), authority, authority FROM mflupload WHERE authority IS NOT NULL AND authority <> ''`,
		`INSERT INTO admin_units (name,mfl_uid,level_id,path) SELECT 'StatGate Field Operations','sg-root',id,'/sg-root/' FROM admin_level WHERE level_number=1`,
		`INSERT INTO admin_units (name,mfl_uid,level_id,parent_id,path) SELECT DISTINCT o.region,o.region_uid,l.id,r.id,r.path || o.region_uid || '/' FROM orgunits_uploads o JOIN admin_level l ON l.level_number=2 JOIN admin_units r ON r.mfl_uid='sg-root' WHERE o.region_uid IS NOT NULL AND o.region_uid<>'' ON CONFLICT (mfl_uid) DO NOTHING`,
		`INSERT INTO admin_units (name,mfl_uid,level_id,parent_id,path) SELECT DISTINCT o.district,o.district_uid,l.id,p.id,p.path || o.district_uid || '/' FROM orgunits_uploads o JOIN admin_level l ON l.level_number=3 JOIN admin_units p ON p.mfl_uid=o.region_uid WHERE o.district_uid IS NOT NULL AND o.district_uid<>'' ON CONFLICT (mfl_uid) DO NOTHING`,
		`INSERT INTO admin_units (name,mfl_uid,level_id,parent_id,path) SELECT DISTINCT o.municipality,o.municipality_uid,l.id,p.id,p.path || o.municipality_uid || '/' FROM orgunits_uploads o JOIN admin_level l ON l.level_number=4 JOIN admin_units p ON p.mfl_uid=o.district_uid WHERE o.municipality_uid IS NOT NULL AND o.municipality_uid<>'' ON CONFLICT (mfl_uid) DO NOTHING`,
		`INSERT INTO admin_units (name,mfl_uid,level_id,parent_id,path) SELECT DISTINCT o.subcounty,o.subcounty_uid,l.id,p.id,p.path || o.subcounty_uid || '/' FROM orgunits_uploads o JOIN admin_level l ON l.level_number=5 JOIN admin_units p ON p.mfl_uid=o.municipality_uid WHERE o.subcounty_uid IS NOT NULL AND o.subcounty_uid<>'' ON CONFLICT (mfl_uid) DO NOTHING`,
		`INSERT INTO admin_units (name,mfl_uid,level_id,parent_id,path) SELECT m.name,m.uid,l.id,p.id,p.path || m.uid || '/' FROM mflupload m JOIN admin_level l ON l.level_number=6 JOIN admin_units p ON p.mfl_uid=m.subcounty_uid ON CONFLICT (mfl_uid) DO UPDATE SET name=EXCLUDED.name, "updatedAt"=NOW()`,
		`INSERT INTO facilities (identifier,mfl_uid,name,short_name,historical_id,admin_unit_id,level,ownership,authority,status,reporting,longitude,latitude) SELECT m.nhfrid,m.uid,m.name,m.shortname,m.uid,u.id,l.mfl_uid,o.mfl_uid,a.mfl_uid,m.status,LOWER(m.report) IN ('reporting','yes','true','1'),NULLIF(m.longtitude,'')::numeric,NULLIF(m.latitude,'')::numeric FROM mflupload m JOIN admin_units u ON u.mfl_uid=m.uid LEFT JOIN "level" l ON l.code=m.hflevel LEFT JOIN ownership o ON o.code=m.ownership LEFT JOIN authority a ON a.code=m.authority`,
		`CREATE OR REPLACE VIEW public.mfl_details AS SELECT f.id,f.identifier,f.name,f.mfl_uid,f.short_name,f.historical_id,r.name AS region,d.name AS district,sc.name AS subcounty,NULL::text AS parish,NULL::text AS village,f.level,f.ownership,f.authority,f.status,f.reporting,f.licensed,f.address,f.contact_personemail,f.contact_personmobile,f.contact_personname,f.contact_persontitle,f.longitude,f.latitude,f.opening_date,f.closing_date,f.bed_capacity,f.services,f."createdAt",f."updatedAt" FROM facilities f JOIN admin_units station ON station.id=f.admin_unit_id LEFT JOIN admin_units sc ON sc.id=station.parent_id LEFT JOIN admin_units municipality ON municipality.id=sc.parent_id LEFT JOIN admin_units d ON d.id=municipality.parent_id LEFT JOIN admin_units r ON r.id=d.parent_id`,
		`CREATE OR REPLACE VIEW public.orgunits AS SELECT station.id AS admin_unit_id,station.mfl_uid,station.name,station.path,station_level.level_number AS level,root.name AS national_name,r.name AS region_name,d.name AS district_city_name,m.name AS dlg_municipality_name,sc.name AS subcounty_division_name,f.identifier,f.short_name,f.level AS station_tier,f.ownership,f.authority,f.status,f.reporting,f.longitude,f.latitude FROM admin_units station JOIN admin_level station_level ON station_level.id=station.level_id LEFT JOIN admin_units sc ON sc.id=station.parent_id LEFT JOIN admin_units m ON m.id=sc.parent_id LEFT JOIN admin_units d ON d.id=m.parent_id LEFT JOIN admin_units r ON r.id=d.parent_id LEFT JOIN admin_units root ON root.mfl_uid='sg-root' LEFT JOIN facilities f ON f.admin_unit_id=station.id`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func loadCSV(db *sql.DB, table, filename string) error {
	path := filepath.Clean(filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\ufeff")))
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read header from %s: %w", path, err)
	}
	for i := range header {
		header[i] = strings.Trim(header[i], "\ufeff")
	}
	columnTypes, err := getColumnTypes(db, table)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("TRUNCATE TABLE public." + table); err != nil {
		return fmt.Errorf("clear %s: %w", table, err)
	}
	stmt, err := tx.Prepare(pq.CopyIn(table, header...))
	if err != nil {
		return fmt.Errorf("prepare copy for %s: %w", table, err)
	}
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			stmt.Close()
			return fmt.Errorf("read %s row %d: %w", filename, count+2, err)
		}
		values := make([]any, len(record))
		for i := range record {
			if record[i] == "" && isNumericColumn(columnTypes[header[i]]) {
				values[i] = nil
			} else {
				values[i] = record[i]
			}
		}
		if _, err := stmt.Exec(values...); err != nil {
			stmt.Close()
			return fmt.Errorf("copy %s row %d: %w", filename, count+2, err)
		}
		count++
	}
	if _, err := stmt.Exec(); err != nil {
		stmt.Close()
		return fmt.Errorf("finalize copy for %s: %w", table, err)
	}
	if err := stmt.Close(); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("%s: loaded %d rows\n", table, count)
	return nil
}

func getColumnTypes(db *sql.DB, table string) (map[string]string, error) {
	rows, err := db.Query(`SELECT column_name, data_type FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	types := map[string]string{}
	for rows.Next() {
		var name, dataType string
		if err := rows.Scan(&name, &dataType); err != nil {
			return nil, err
		}
		types[name] = dataType
	}
	return types, rows.Err()
}

func isNumericColumn(dataType string) bool {
	switch dataType {
	case "smallint", "integer", "bigint", "numeric", "real", "double precision", "decimal":
		return true
	default:
		return false
	}
}
