package main

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// ── types ─────────────────────────────────────────────────────────────────────

// MVDDLInfo holds column and UID mapping data parsed from a DDL SQL file.
type MVDDLInfo struct {
	SourceFile       string              // relative path to the SQL file
	Columns          map[string]bool     // all column names present in the MV
	UIDToColumns     map[string][]string // uid → []column_name (one uid may alias multiple cols)
	ColumnToUID      map[string]string   // column_name → uid (reverse of UIDToColumns primary entry)
	UIDToDescription map[string]string   // uid → human-readable label (from comment block "| description")
}

// ── cache ─────────────────────────────────────────────────────────────────────

var (
	ddlSchemaCache     map[string]*MVDDLInfo // key: "schema.table" (lower-case)
	ddlSchemaCacheOnce sync.Once
	ddlSchemaCacheMu   sync.RWMutex
)

// GetDDLSchemaCache returns the lazily-initialised DDL schema cache.
// On first call it walks database/ and configs/ directories for *.sql files.
func GetDDLSchemaCache() map[string]*MVDDLInfo {
	ddlSchemaCacheOnce.Do(func() {
		cache := buildDDLSchemaCache()
		ddlSchemaCacheMu.Lock()
		ddlSchemaCache = cache
		ddlSchemaCacheMu.Unlock()
	})
	ddlSchemaCacheMu.RLock()
	defer ddlSchemaCacheMu.RUnlock()
	return ddlSchemaCache
}

func buildDDLSchemaCache() map[string]*MVDDLInfo {
	cache := make(map[string]*MVDDLInfo)
	for _, root := range []string{"database", "configs", "local_scripts_docs"} {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if strings.EqualFold(filepath.Ext(path), ".sql") {
				parseSQLFileIntoDDLCache(path, cache)
			}
			return nil
		})
	}
	totalCols := 0
	for _, v := range cache {
		totalCols += len(v.Columns)
	}
	logInfo("DDL schema cache: %d views, %d total columns", len(cache), totalCols)
	return cache
}

// ── parsing ───────────────────────────────────────────────────────────────────

// Regexes used by the DDL parser.
var (
	// commentMVSectionRe detects alias-mapping section headers in block comments:
	//   "---- mv_example_table alias mapping ----"
	commentMVSectionRe = regexp.MustCompile(`(?i)-{3,}\s*(mv_[a-z0-9_]+)\s+alias\s+mapping`)

	// commentAliasLineRe captures "- column_name => UID | description" lines.
	// UIDs are exactly 11 alphanumeric characters (canonical historical standard).
	// Group 1: column alias, Group 2: UID, Group 3 (optional): description after "|".
	commentAliasLineRe = regexp.MustCompile(`^\s*-\s+(\w+)\s*=>\s*([A-Za-z0-9]{11})\b(?:[^|]*\|\s*(.+))?`)

	// createMVRe detects CREATE MATERIALIZED VIEW schema.table statements.
	createMVRe = regexp.MustCompile(`(?i)CREATE\s+MATERIALIZED\s+VIEW\s+(\w+)\.(\w+)`)

	// castAliasRe captures ")::type AS column_name" patterns (aggregation columns).
	castAliasRe = regexp.MustCompile(`\)::(?:numeric|text|integer|bigint|boolean|varchar)\s+AS\s+([a-z][a-z0-9_]+)`)

	// selectAsRe captures plain "AS column_name" at end of SELECT list items.
	// Anchored to end-of-line or followed by comma/whitespace to avoid CTE name clashes.
	selectAsRe = regexp.MustCompile(`\bAS\s+([a-z][a-z0-9_]+)\s*(?:[,;)]|$)`)
)

// sqlKeywordsInAS is a deny-list of SQL identifiers that follow AS but are not
// MV column names (mostly SQL clause keywords and common table aliases).
var sqlKeywordsInAS = map[string]bool{
	"select": true, "from": true, "where": true, "join": true, "on": true,
	"and": true, "or": true, "not": true, "in": true, "is": true, "null": true,
	"true": true, "false": true, "case": true, "when": true, "then": true,
	"else": true, "end": true, "order": true, "by": true, "group": true,
	"having": true, "limit": true, "with": true, "data": true, "table": true,
	"view": true, "index": true, "create": true, "unique": true, "values": true,
	"set": true, "into": true, "inner": true, "outer": true, "left": true,
	"right": true, "full": true, "cross": true, "union": true, "all": true,
	"distinct": true, "asc": true, "desc": true, "between": true, "like": true,
	"exists": true, "any": true, "some": true, "coalesce": true, "nullif": true,
	"extract": true, "over": true, "partition": true, "window": true,
	"row": true, "rows": true, "following": true, "preceding": true,
	"unbounded": true, "current": true, "no": true, "other": true,
	// common short table aliases
	"a": true, "b": true, "d": true, "l": true, "m": true, "s": true,
	"p": true, "c": true, "r": true, "x": true, "dp": true, "agg": true,
}

// parseSQLFileIntoDDLCache reads a SQL file and populates cache.
//
// Two extraction passes are performed:
//  1. Comment-block alias sections:  "---- mv_NAME alias mapping ---"
//     followed by "- column_name => UID | ..." lines.
//  2. DDL-level extraction: after each "CREATE MATERIALIZED VIEW schema.table",
//     collect ")::type AS column_name" cast patterns.
func parseSQLFileIntoDDLCache(path string, cache map[string]*MVDDLInfo) {
	f, err := os.Open(path)
	if err != nil {
		logWarn("DDL cache: cannot open %s: %v", path, err)
		return
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		logWarn("DDL cache: read error %s: %v", path, err)
		return
	}

	// ── Pass 1: comment-block alias sections ──────────────────────────────────
	// Collect per-MV short-name mappings from the block-comment header.
	type commentSection struct {
		cols    map[string]bool
		uidCols map[string][]string
		colUID  map[string]string // column_name → uid
		uidDesc map[string]string // uid → description (first occurrence wins)
	}
	sections := make(map[string]*commentSection) // key: short table name
	var curSec *commentSection

	for _, line := range lines {
		if m := commentMVSectionRe.FindStringSubmatch(line); m != nil {
			name := strings.ToLower(m[1])
			if sections[name] == nil {
				sections[name] = &commentSection{
					cols:    make(map[string]bool),
					uidCols: make(map[string][]string),
					colUID:  make(map[string]string),
					uidDesc: make(map[string]string),
				}
			}
			curSec = sections[name]
			continue
		}
		if curSec != nil {
			if m := commentAliasLineRe.FindStringSubmatch(line); m != nil {
				col := strings.ToLower(m[1])
				uid := m[2]
				curSec.cols[col] = true
				// A single UID may map to multiple column aliases (e.g. gHjJrtw9vQX
				// maps to both case_deaths and confirmed_deaths).
				curSec.uidCols[uid] = ddlAppendUniq(curSec.uidCols[uid], col)
				// col → uid reverse map (first occurrence wins for multi-alias UIDs)
				if _, exists := curSec.colUID[col]; !exists {
					curSec.colUID[col] = uid
				}
				// description after "|"
				if len(m) >= 4 && strings.TrimSpace(m[3]) != "" {
					if _, exists := curSec.uidDesc[uid]; !exists {
						curSec.uidDesc[uid] = strings.TrimSpace(m[3])
					}
				}
			}
		}
	}

	// ── Pass 2: CREATE MATERIALIZED VIEW + DDL column extraction ─────────────
	var curInfo *MVDDLInfo

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)

		if m := createMVRe.FindStringSubmatch(rawLine); m != nil {
			schema := strings.ToLower(m[1])
			table := strings.ToLower(m[2])
			fqn := schema + "." + table

			info := &MVDDLInfo{
				SourceFile:       path,
				Columns:          make(map[string]bool),
				UIDToColumns:     make(map[string][]string),
				ColumnToUID:      make(map[string]string),
				UIDToDescription: make(map[string]string),
			}
			// Seed from comment-block data.
			if sec, ok := sections[table]; ok {
				for c := range sec.cols {
					info.Columns[c] = true
				}
				for uid, cols := range sec.uidCols {
					info.UIDToColumns[uid] = append(info.UIDToColumns[uid], cols...)
				}
				for col, uid := range sec.colUID {
					info.ColumnToUID[col] = uid
				}
				for uid, desc := range sec.uidDesc {
					info.UIDToDescription[uid] = desc
				}
			}
			cache[fqn] = info
			curInfo = info
			continue
		}

		if curInfo == nil {
			continue
		}

		lower := strings.ToLower(line)

		// Collect cast-alias column names: ")::numeric AS column_name"
		if m := castAliasRe.FindStringSubmatch(lower); m != nil {
			curInfo.Columns[m[1]] = true
		}

		// Collect general "AS column_name" SELECT-list aliases.
		for _, m := range selectAsRe.FindAllStringSubmatch(lower, -1) {
			col := m[1]
			if !sqlKeywordsInAS[col] && len(col) > 1 {
				curInfo.Columns[col] = true
			}
		}
	}
}

// ddlAppendUniq appends s to slice only if not already present.
func ddlAppendUniq(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}
