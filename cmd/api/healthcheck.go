package main

import (
	"net/http"
)

type healthcheckData struct {
	Status     string         `json:"status"`
	SystemInfo systemInfoData `json:"system_info"`
}

type systemInfoData struct {
	Environment string `json:"environment"`
	Version     string `json:"version"`
}

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	data := healthcheckData{
		Status: "available",
		SystemInfo: systemInfoData{
			Environment: app.cfg.env,
			Version:     version,
		},
	}

	if err := app.writeJSON(w, http.StatusOK, data, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
