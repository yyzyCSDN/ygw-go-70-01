package main

import (
	"pvcontrol/internal/alarm"
	"pvcontrol/internal/comm"
	"pvcontrol/internal/grid"
	"pvcontrol/internal/inverter"
	"pvcontrol/internal/param"
	"pvcontrol/internal/power"
)

type Runtime struct {
	Table     *inverter.Table
	Catalog   *inverter.Catalog
	Comm      *comm.Client
	Hub       *comm.Hub
	Grid      *grid.Manager
	Power     *power.Manager
	Param     *param.Manager
	ParamCatalog *param.Catalog
	Alarm     *alarm.Manager
	Station   *inverter.Station
	Scheduler *inverter.Scheduler
	Forecast  *power.Forecast
	Ramp      *power.RampLimiter
	Island    *grid.IslandGuard
	Escalator *alarm.Escalator
}

func BuildRuntime() *Runtime {
	table := inverter.NewTable()
	catalog := inverter.DefaultCatalog()
	_ = table.Register(&inverter.Inverter{ID: "inv-01", Serial: "PV1A0001", Model: "PV-100K"})
	_ = table.Register(&inverter.Inverter{ID: "inv-02", Serial: "PV1B0002", Model: "PV-50K"})
	_ = table.Register(&inverter.Inverter{ID: "inv-03", Serial: "PV1C0003", Model: "PV-200K"})
	table.FreezeRestoreState()
	client := comm.NewClient(table, comm.NewSimTransport(table))
	hub := comm.NewHub()
	_ = hub.Add("main", client)
	gridMgr := grid.NewManager(table, client)
	powerMgr := power.NewManager(350000, client)
	paramCatalog := param.DefaultCatalog(catalog)
	paramMgr := param.NewManager(table, paramCatalog, client)
	alarmMgr := alarm.NewManager(table)
	station := inverter.NewStation(table, client)
	scheduler := inverter.NewScheduler(table, client)
	forecast := power.NewForecast(10)
	ramp := power.NewRampLimiter(20000)
	islandGuard := grid.NewIslandGuard(49.5, 50.5)
	escalator := alarm.NewEscalator(nil)
	return &Runtime{
		Table:     table,
		Catalog:   catalog,
		Comm:      client,
		Hub:       hub,
		Grid:      gridMgr,
		Power:     powerMgr,
		Param:     paramMgr,
		ParamCatalog: paramCatalog,
		Alarm:     alarmMgr,
		Station:   station,
		Scheduler: scheduler,
		Forecast:  forecast,
		Ramp:      ramp,
		Island:    islandGuard,
		Escalator: escalator,
	}
}
