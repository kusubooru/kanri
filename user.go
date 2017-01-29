package main

import (
	"fmt"
	"net/http"
	"strconv"
)

func (app *App) serveUserFind(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("userID")
	if id == "" {
		app.render(w, userFindTmpl, nil)
		return
	}

	userID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad user ID: %v", err), http.StatusBadRequest)
		return
	}

	u, err := app.Shimmie.GetUser(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve user: %v", err), http.StatusInternalServerError)
		return
	}

	app.render(w, userFindTmpl, u)
}
