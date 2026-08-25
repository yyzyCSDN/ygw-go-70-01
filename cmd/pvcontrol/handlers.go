package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"pvcontrol/internal/alarm"
	"pvcontrol/internal/grid"
	"pvcontrol/internal/inverter"
	"pvcontrol/internal/param"
	"pvcontrol/internal/power"
)

type inverterView struct {
	ID        string `json:"id"`
	Serial    string `json:"serial"`
	Model     string `json:"model"`
	State     string `json:"state"`
	GridState string `json:"grid_state"`
	OutputW   int    `json:"output_w"`
	ParamRev  int    `json:"param_rev"`
}

func (s *Server) handleInverters(w http.ResponseWriter, r *http.Request) {
	out := make([]inverterView, 0)
	for _, inv := range s.rt.Table.Snapshot() {
		gridState, _ := s.rt.Table.GridStateOf(inv.ID)
		out = append(out, inverterView{
			ID:        inv.ID,
			Serial:    inv.Serial,
			Model:     inv.Model,
			State:     string(inv.State),
			GridState: string(gridState),
			OutputW:   inv.OutputW,
			ParamRev:  inv.ParamRev,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

type irradianceRequest struct {
	Irradiance int `json:"irradiance"`
}

func (s *Server) handleIrradiance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req irradianceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	s.rt.Forecast.Add(req.Irradiance)
	s.rt.Ramp.Reset(0)
	if err := s.rt.Power.UpdateIrradiance(req.Irradiance); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"irradiance": req.Irradiance,
		"target_w":   s.rt.Power.Target(),
		"ramp_w":     s.rt.Ramp.Current(),
	})
}

func (s *Server) handleForecast(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"average": s.rt.Forecast.Average(),
		"latest":  s.rt.Forecast.Latest(),
		"count":   s.rt.Forecast.Count(),
	})
}

func (s *Server) handleAlarms(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			Action     string `json:"action"`
			InverterID string `json:"inverter_id"`
			Kind       string `json:"kind"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
			return
		}
		var err error
		switch req.Action {
		case "raise":
			err = s.rt.Alarm.Raise(req.InverterID, req.Kind)
		case "resolve":
			err = s.rt.Alarm.Resolve(req.InverterID)
		case "rebuild":
			err = s.rt.Alarm.RebuildAfterRestart()
		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown alarm action"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
			return
		}
	}
	active := s.rt.Alarm.Active()
	severity := s.rt.Escalator.Level(alarm.SeverityIndex(len(active), 5))
	writeJSON(w, http.StatusOK, map[string]any{
		"count":    s.rt.Alarm.ActiveCount(),
		"alarms":   active,
		"severity": severity,
		"levels":   s.rt.Escalator.Count(),
		"rebuild_count": s.rt.Alarm.LastRebuildCount(),
	})
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	if err := s.rt.Grid.Connect(id); err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "grid_state": s.rt.Grid.StateOf(id)})
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	if err := s.rt.Station.Start(id); err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "state": "running"})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	if err := s.rt.Station.Stop(id); err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "state": "stopped"})
}

func (s *Server) handleRecover(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	if err := s.rt.Station.Recover(id); err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "state": "running"})
}

func (s *Server) handlePatrol(w http.ResponseWriter, r *http.Request) {
	invs := s.rt.Table.Snapshot()
	ids := make([]string, 0, len(invs))
	for _, inv := range invs {
		ids = append(ids, inv.ID)
	}
	if err := s.rt.Comm.Patrol(ids); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"patrolled": len(ids),
		"sessions":  s.rt.Comm.SessionCount(),
	})
}

type paramRequest struct {
	InverterID    string `json:"inverter_id"`
	LimitPowerW   int    `json:"limit_power_w"`
	ReactiveLimit int    `json:"reactive_limit"`
	RampRate      int    `json:"ramp_rate"`
	VoltRef       int    `json:"volt_ref"`
	FreqRef       int    `json:"freq_ref"`
}

func (s *Server) handleParam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req paramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	inv, err := s.rt.Table.Get(req.InverterID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	next := inverter.ParamSet{
		Model:         inv.Model,
		Rev:           inv.ParamRev + 1,
		LimitPowerW:   req.LimitPowerW,
		ReactiveLimit: req.ReactiveLimit,
		RampRate:      req.RampRate,
		VoltRef:       req.VoltRef,
		FreqRef:       req.FreqRef,
	}
	if err := s.rt.Param.Deliver(req.InverterID, next); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	changes := param.Diff(inv.Params, next)
	delivered, _ := s.rt.Param.Delivered(req.InverterID)
	appliedModel := s.rt.Param.AppliedModel(req.InverterID)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":            req.InverterID,
		"rev":           next.Rev,
		"changed":       changes,
		"delivered_rev": delivered.Rev,
		"applied_model": appliedModel,
		"applied_at":    inv.ParamAppliedAt,
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := inverter.CollectStats(s.rt.Table, s.rt.Catalog)
	writeJSON(w, http.StatusOK, map[string]any{
		"total":       stats.Total,
		"running":     stats.Running,
		"fault":       stats.Fault,
		"stopped":     stats.Stopped,
		"derating":    stats.Derating,
		"starting":    stats.Starting,
		"output_w":    stats.OutputW,
		"capacity_w":  stats.Capacity,
		"available_w": inverter.AvailableCapacity(stats),
		"ratio":       inverter.OutputRatio(stats),
	})
}

type scheduleRequest struct {
	Order      []string `json:"order"`
	Action     string   `json:"action"`
	InverterID string   `json:"inverter_id"`
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req scheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	var err error
	switch req.Action {
	case "", "start":
		err = s.rt.Scheduler.SetOrder(req.Order)
		if err == nil {
			err = s.rt.Scheduler.StartAll()
		}
	case "stop":
		err = s.rt.Scheduler.StopAll()
	case "start_one":
		err = s.rt.Scheduler.StartOne(req.InverterID)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown schedule action"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"action": req.Action,
		"count":  s.rt.Scheduler.Count(),
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	sent, failed, retries, frames := s.rt.Comm.Metrics().Snapshot()
	writeJSON(w, http.StatusOK, map[string]any{
		"sent":        sent,
		"failed":      failed,
		"retries":     retries,
		"frames":      frames,
		"sessions":    s.rt.Comm.SessionCount(),
		"hub_clients": s.rt.Hub.ClientNames(),
		"hub_count":   s.rt.Hub.Count(),
	})
}

type islandRequest struct {
	InverterID string  `json:"inverter_id"`
	Frequency  float64 `json:"frequency"`
}

func (s *Server) handleIsland(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req islandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	tripped := s.rt.Island.Trip(req.Frequency)
	if tripped {
		if err := s.rt.Grid.MarkIsland(req.InverterID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
	} else {
		_ = s.rt.Grid.RestoreFromIsland(req.InverterID)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"islanded":  tripped,
		"frequency": req.Frequency,
		"clear":     s.rt.Island.Clear(req.Frequency),
	})
}

type derateRequest struct {
	InverterID string `json:"inverter_id"`
	Percent    int    `json:"percent"`
}

func (s *Server) handleDerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req derateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	inv, err := s.rt.Table.Get(req.InverterID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		return
	}
	model, err := s.rt.Catalog.Get(inv.Model)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	target := power.DerateTarget(model.RatedPowerW, req.Percent)
	target = s.rt.Ramp.Step(target)
	if err := s.rt.Station.LimitPower(req.InverterID, target); err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       req.InverterID,
		"target_w": target,
		"ramp_w":   s.rt.Ramp.Current(),
	})
}

type ingestRequest struct {
	Data string `json:"data"`
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	raw, err := hex.DecodeString(req.Data)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid hex"})
		return
	}
	frame, err := s.rt.Hub.IngestWith("main", raw)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"serial": frame.Serial, "kind": frame.Kind})
}

type batchRequest struct {
	InverterID string `json:"inverter_id"`
	Kind       string `json:"kind"`
	Count      int    `json:"count"`
}

func (s *Server) handleBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req batchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	if req.Count < 1 || req.Count > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "count out of range"})
		return
	}
	payloads := make([][]byte, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		payloads = append(payloads, inverter.EncodePower(req.Count*1000+i))
	}
	if _, err := s.rt.Comm.EnqueueBatch(req.InverterID, req.Kind, payloads); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	processed, err := s.rt.Comm.ProcessPending(req.InverterID)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "processed": processed})
		return
	}
	statusBatch := []inverter.QueuedMessage{{InverterID: req.InverterID, Seq: 1}}
	statuses := s.rt.Table.MessageStatusesFor(statusBatch)
	writeJSON(w, http.StatusOK, map[string]any{"processed": processed, "statuses": statuses})
}

func (s *Server) handlePlan(w http.ResponseWriter, r *http.Request) {
	irradiance := 0
	if raw := r.URL.Query().Get("irradiance"); raw != "" {
		_, _ = fmt.Sscanf(raw, "%d", &irradiance)
	}
	invs := make([]power.PlantInverter, 0)
	for _, inv := range s.rt.Table.Snapshot() {
		model, err := s.rt.Catalog.Get(inv.Model)
		if err != nil {
			continue
		}
		invs = append(invs, power.PlantInverter{
			ID:          inv.ID,
			Model:       inv.Model,
			RatedPowerW: model.RatedPowerW,
			MinPowerW:   model.MinPowerW,
		})
	}
	plan := power.BuildPlan(irradiance, power.PlantCapacity(invs), invs)
	writeJSON(w, http.StatusOK, map[string]any{
		"irradiance":  irradiance,
		"target_w":    plan.TargetW,
		"assignments": plan.Assignments,
		"by_model":    power.PlanByModel(invs),
	})
}

func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	models := s.rt.Catalog.List()
	writeJSON(w, http.StatusOK, map[string]any{
		"models":      models,
		"count":       len(models),
		"has_pv_100k": s.rt.Catalog.Has("PV-100K"),
	})
}

func (s *Server) handleGridOverview(w http.ResponseWriter, r *http.Request) {
	members := s.rt.Grid.Members()
	states := make(map[string]string, len(members))
	errors := make(map[string]string, len(members))
	sequences := make(map[string]int, len(members))
	signatures := make(map[string]uint64, len(members))
	for _, id := range members {
		states[id] = string(s.rt.Grid.StateOf(id))
		if err := s.rt.Grid.LastError(id); err != nil {
			errors[id] = err.Error()
		}
		signatures[id] = s.rt.Comm.LastSequenceSignature(id)
		if seq, err := s.rt.Grid.SequenceOf(id); err == nil {
			sequences[id] = seq.StepCount()
			if seq.HasStep(id) {
				sequences[id]++
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"members":      members,
		"member_count": s.rt.Grid.MemberCount(),
		"states":       states,
		"last_errors":  errors,
		"step_counts":  sequences,
		"signatures":   signatures,
	})
}

type gridActionRequest struct {
	InverterID string `json:"inverter_id"`
	State      string `json:"state"`
}

func (s *Server) handleGridJoin(w http.ResponseWriter, r *http.Request) {
	var req gridActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	if err := s.rt.Grid.Join(req.InverterID); err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"joined": req.InverterID})
}

func (s *Server) handleGridSwitch(w http.ResponseWriter, r *http.Request) {
	var req gridActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	target := grid.GridState(req.State)
	if err := s.rt.Grid.SwitchTo(req.InverterID, target); err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": req.InverterID, "state": target})
}

func (s *Server) handleGridResync(w http.ResponseWriter, r *http.Request) {
	var req gridActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	if err := s.rt.Grid.Resync(req.InverterID); err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"resynced": req.InverterID})
}

func (s *Server) handleGridRebuild(w http.ResponseWriter, r *http.Request) {
	if err := s.rt.Grid.Rebuild(); err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rebuilt": s.rt.Grid.MemberCount()})
}

func (s *Server) handleGridDisconnect(w http.ResponseWriter, r *http.Request) {
	var req gridActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	if err := s.rt.Grid.Disconnect(req.InverterID); err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"disconnected": req.InverterID})
}

func (s *Server) handleParamPlan(w http.ResponseWriter, r *http.Request) {
	invID := r.URL.Query().Get("inverter_id")
	plan, err := s.rt.Param.PlanFor(invID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"inverter_id": invID, "plan": plan})
}

func (s *Server) handleParamProfiles(w http.ResponseWriter, r *http.Request) {
	profiles := param.Summarize(s.rt.Table, s.rt.ParamCatalog)
	writeJSON(w, http.StatusOK, map[string]any{"profiles": profiles})
}

func (s *Server) handleTick(w http.ResponseWriter, r *http.Request) {
	if err := s.rt.Power.Tick(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"target_w": s.rt.Power.Target()})
}

func (s *Server) handleOpsSnapshot(w http.ResponseWriter, r *http.Request) {
	rows := make([]map[string]any, 0)
	for _, inv := range s.rt.Table.Snapshot() {
		gridState, _ := s.rt.Table.GridStateOf(inv.ID)
		leases, _ := s.rt.Table.ActiveLeases(inv.ID)
		statuses := s.rt.Table.MessageStatusesFor([]inverter.QueuedMessage{{InverterID: inv.ID, Seq: 1}})
		rows = append(rows, map[string]any{
			"id":            inv.ID,
			"grid_state":    gridState,
			"active_leases": leases,
			"message_count": len(statuses),
			"last_grid_err": inv.LastGridErr,
			"grid_err_count": inv.GridErrCount,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"restore_count": len(s.rt.Table.RestoreState()),
		"inverters":     rows,
	})
}

type replaceRequest struct {
	InverterID string `json:"inverter_id"`
	Serial     string `json:"serial"`
	Model      string `json:"model"`
}

func (s *Server) handleReplace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req replaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid body"})
		return
	}
	if err := s.rt.Table.Replace(req.InverterID, req.Serial, req.Model); err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	model, _ := s.rt.Table.ModelOf(req.InverterID)
	inv, _ := s.rt.Table.Get(req.InverterID)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":        req.InverterID,
		"model":     model,
		"model_rev": inv.ModelRev,
	})
}
