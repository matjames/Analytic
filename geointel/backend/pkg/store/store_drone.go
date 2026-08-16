package store

import (
	"context"
	"database/sql"
	"time"

	"geointel/pkg/model"
)

// ─── Drones (P44) ────────────────────────────────────────────────────────────

func CreateDrone(ctx context.Context, d *model.Drone) error {
	d.ID = newID()
	d.CreatedAt = nowT()
	if d.Status == "" {
		d.Status = "grounded"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO drones (id, tenant_id, name, model, serial, status, capabilities, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		d.ID, d.TenantID, d.Name, d.Model, d.Serial, d.Status, jsonB(d.Capabilities), d.CreatedAt)
	return err
}

const droneCols = `id, tenant_id, name, model, serial, status, capabilities, created_at`

func ListDrones(ctx context.Context, tenantID string) ([]model.Drone, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+droneCols+` FROM drones WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Drone
	for rows.Next() {
		var d model.Drone
		var cap []byte
		if err := rows.Scan(&d.ID, &d.TenantID, &d.Name, &d.Model, &d.Serial, &d.Status, &cap, &d.CreatedAt); err != nil {
			return nil, err
		}
		jsonUnmarshal(cap, &d.Capabilities)
		out = append(out, d)
	}
	return out, rows.Err()
}

func GetDrone(ctx context.Context, id string) (*model.Drone, error) {
	var d model.Drone
	var cap []byte
	err := db.QueryRowContext(ctx, `SELECT `+droneCols+` FROM drones WHERE id=$1`, id).
		Scan(&d.ID, &d.TenantID, &d.Name, &d.Model, &d.Serial, &d.Status, &cap, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	jsonUnmarshal(cap, &d.Capabilities)
	return &d, nil
}

func UpdateDroneStatus(ctx context.Context, id, status string) error {
	_, err := db.ExecContext(ctx, `UPDATE drones SET status=$2 WHERE id=$1`, id, status)
	return err
}

// ─── Flight Plans (P44) ──────────────────────────────────────────────────────

func CreateFlightPlan(ctx context.Context, p *model.FlightPlan) error {
	p.ID = newID()
	p.CreatedAt = nowT()
	if p.Status == "" {
		p.Status = "active"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO flight_plans (id, tenant_id, drone_id, name, waypoints, altitude, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		p.ID, p.TenantID, p.DroneID, p.Name, jsonB(p.Waypoints), p.Altitude, p.Status, p.CreatedAt)
	return err
}

func ListFlightPlans(ctx context.Context, tenantID string) ([]model.FlightPlan, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, tenant_id, drone_id, name, waypoints, altitude, status, created_at FROM flight_plans WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.FlightPlan
	for rows.Next() {
		var p model.FlightPlan
		var wp []byte
		if err := rows.Scan(&p.ID, &p.TenantID, &p.DroneID, &p.Name, &wp, &p.Altitude, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		jsonUnmarshal(wp, &p.Waypoints)
		out = append(out, p)
	}
	return out, rows.Err()
}

var _ = sql.ErrNoRows

// ─── Flights & Telemetry (P44) ───────────────────────────────────────────────

func CreateFlight(ctx context.Context, f *model.Flight) error {
	f.ID = newID()
	f.CreatedAt = nowT()
	if f.Status == "" {
		f.Status = "planned"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO flights (id, tenant_id, drone_id, plan_id, status, summary, started_at, ended_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		f.ID, f.TenantID, f.DroneID, f.PlanID, f.Status, jsonB(f.Summary), f.StartedAt, f.EndedAt, f.CreatedAt)
	return err
}

func scanFlight(row row) (*model.Flight, error) {
	var f model.Flight
	var summary []byte
	var start, end sql.NullTime
	if err := row.Scan(&f.ID, &f.TenantID, &f.DroneID, &f.PlanID, &f.Status, &summary, &start, &end, &f.CreatedAt); err != nil {
		return nil, err
	}
	jsonUnmarshal(summary, &f.Summary)
	if start.Valid {
		f.StartedAt = &start.Time
	}
	if end.Valid {
		f.EndedAt = &end.Time
	}
	return &f, nil
}

const flightCols = `id, tenant_id, drone_id, plan_id, status, summary, started_at, ended_at, created_at`

func ListFlights(ctx context.Context, tenantID, droneID string) ([]model.Flight, error) {
	query := `SELECT ` + flightCols + ` FROM flights WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if droneID != "" {
		query += ` AND drone_id=$2`
		args = append(args, droneID)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Flight
	for rows.Next() {
		f, err := scanFlight(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func GetFlight(ctx context.Context, id string) (*model.Flight, error) {
	return scanFlight(db.QueryRowContext(ctx, `SELECT `+flightCols+` FROM flights WHERE id=$1`, id))
}

// CompleteFlight ends a flight with a status and summary.
func CompleteFlight(ctx context.Context, id, status string, summary map[string]interface{}) (*model.Flight, error) {
	f, err := GetFlight(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	f.Status = status
	f.EndedAt = &now
	f.Summary = summary
	_, err = db.ExecContext(ctx, `UPDATE flights SET status=$2, summary=$3, ended_at=$4 WHERE id=$1`,
		id, status, jsonB(summary), now)
	return f, err
}

func StartFlight(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := db.ExecContext(ctx, `UPDATE flights SET status='in_flight', started_at=$2 WHERE id=$1`, id, now)
	return err
}

func ListTelemetry(ctx context.Context, flightID string) ([]model.TelemetryPoint, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, drone_id, flight_id, ts, lat, lng, alt, battery, speed
		FROM telemetry WHERE flight_id=$1 ORDER BY ts`, flightID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.TelemetryPoint
	for rows.Next() {
		var tp model.TelemetryPoint
		if err := rows.Scan(&tp.ID, &tp.TenantID, &tp.DroneID, &tp.FlightID, &tp.TS, &tp.Lat, &tp.Lng, &tp.Alt, &tp.Battery, &tp.Speed); err != nil {
			return nil, err
		}
		out = append(out, tp)
	}
	return out, rows.Err()
}

func InsertTelemetry(ctx context.Context, tp *model.TelemetryPoint) error {
	tp.ID = newID()
	if tp.TS.IsZero() {
		tp.TS = time.Now().UTC()
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO telemetry (id, tenant_id, drone_id, flight_id, ts, lat, lng, alt, battery, speed)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		tp.ID, tp.TenantID, tp.DroneID, tp.FlightID, tp.TS, tp.Lat, tp.Lng, tp.Alt, tp.Battery, tp.Speed)
	return err
}