package api

import (
	"encoding/json"
	"net/http"

	"geointel/pkg/model"
	"geointel/pkg/store"
)

// ─── Drones (P44) ────────────────────────────────────────────────────────────

func ListDronesHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListDrones(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateDroneHandler(w http.ResponseWriter, r *http.Request) {
	var d model.Drone
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if d.Name == "" || d.Model == "" {
		writeError(w, http.StatusBadRequest, "name and model are required")
		return
	}
	d.TenantID = actorTenant(r)
	if err := store.CreateDrone(r.Context(), &d); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func GetDroneHandler(w http.ResponseWriter, r *http.Request) {
	d, err := store.GetDrone(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "drone not found")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func DroneStatusHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := store.GetDrone(r.Context(), varsOf(r)["id"]); err != nil {
		writeError(w, http.StatusNotFound, "drone not found")
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if body.Status == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}
	if err := store.UpdateDroneStatus(r.Context(), varsOf(r)["id"], body.Status); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": body.Status})
}

// ─── Flight Plans (P44) ──────────────────────────────────────────────────────

func ListFlightPlansHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListFlightPlans(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateFlightPlanHandler(w http.ResponseWriter, r *http.Request) {
	var p model.FlightPlan
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if p.Name == "" || p.DroneID == "" {
		writeError(w, http.StatusBadRequest, "name and drone_id are required")
		return
	}
	if _, err := store.GetDrone(r.Context(), p.DroneID); err != nil {
		writeError(w, http.StatusBadRequest, "drone does not exist")
		return
	}
	p.TenantID = actorTenant(r)
	if err := store.CreateFlightPlan(r.Context(), &p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// ─── Flights & Telemetry (P44) ───────────────────────────────────────────────

func ListFlightsHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListFlights(r.Context(), actorTenant(r), r.URL.Query().Get("drone_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func StartFlightHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DroneID string `json:"drone_id"`
		PlanID  string `json:"plan_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if body.DroneID == "" {
		writeError(w, http.StatusBadRequest, "drone_id is required")
		return
	}
	if _, err := store.GetDrone(r.Context(), body.DroneID); err != nil {
		writeError(w, http.StatusBadRequest, "drone does not exist")
		return
	}
	f := model.Flight{TenantID: actorTenant(r), DroneID: body.DroneID, PlanID: body.PlanID, Status: "in_flight"}
	if err := store.CreateFlight(r.Context(), &f); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := store.StartFlight(r.Context(), f.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = store.UpdateDroneStatus(r.Context(), body.DroneID, "in_flight")
	writeJSON(w, http.StatusCreated, f)
}

func FlightTelemetryHandler(w http.ResponseWriter, r *http.Request) {
	flightID := varsOf(r)["id"]
	f, err := store.GetFlight(r.Context(), flightID)
	if err != nil {
		writeError(w, http.StatusNotFound, "flight not found")
		return
	}
	var tp model.TelemetryPoint
	if err := json.NewDecoder(r.Body).Decode(&tp); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	tp.TenantID = actorTenant(r)
	tp.DroneID = f.DroneID
	tp.FlightID = flightID
	if err := store.InsertTelemetry(r.Context(), &tp); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tp)
}

func ListFlightTelemetryHandler(w http.ResponseWriter, r *http.Request) {
	flightID := varsOf(r)["id"]
	if _, err := store.GetFlight(r.Context(), flightID); err != nil {
		writeError(w, http.StatusNotFound, "flight not found")
		return
	}
	items, err := store.ListTelemetry(r.Context(), flightID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// EndFlightHandler completes a mission, computes a summary from telemetry and
// emits drone.flight.completed.
func EndFlightHandler(w http.ResponseWriter, r *http.Request) {
	flightID := varsOf(r)["id"]
	f, err := store.GetFlight(r.Context(), flightID)
	if err != nil {
		writeError(w, http.StatusNotFound, "flight not found")
		return
	}
	telemetry, err := store.ListTelemetry(r.Context(), flightID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	points := len(telemetry)
	distance := 0.0
	for i := 1; i < points; i++ {
		distance += store.HaversineMeters(telemetry[i-1].Lat, telemetry[i-1].Lng, telemetry[i].Lat, telemetry[i].Lng)
	}
	summary := map[string]interface{}{"telemetry_points": points, "ground_distance_m": round2(distance)}
	f, err = store.CompleteFlight(r.Context(), flightID, "completed", summary)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = store.UpdateDroneStatus(r.Context(), f.DroneID, "ready")
	emitEvent(r.Context(), "drone.flight.completed", "flight", f.ID, map[string]interface{}{"drone_id": f.DroneID, "points": points})
	recordAudit(r, "drone.flight.completed", "flight", f.ID, map[string]interface{}{"drone_id": f.DroneID})
	writeJSON(w, http.StatusOK, f)
}