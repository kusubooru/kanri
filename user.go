package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
)

type userFindCtxKey int

const (
	userIDCtxKey userFindCtxKey = iota
	usernameCtxKey
)

func (app *App) serveUserFind(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("userID")
	if id != "" {
		ctx := context.WithValue(r.Context(), userIDCtxKey, id)
		app.serveUserFindByID(w, r.WithContext(ctx))
		return
	}

	username := r.FormValue("username")
	if username != "" {
		ctx := context.WithValue(r.Context(), usernameCtxKey, username)
		app.serveUserFindByName(w, r.WithContext(ctx))
		return
	}

	app.render(w, userFindTmpl, nil)
}

func (app *App) serveUserFindByID(w http.ResponseWriter, r *http.Request) {
	id, ok := r.Context().Value(userIDCtxKey).(string)
	if !ok {
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
		if err == sql.ErrNoRows {
			app.render(w, userFindTmpl, nil)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to retrieve user: %v", err), http.StatusInternalServerError)
		return
	}

	app.render(w, userFindTmpl, u)
}

func (app *App) serveUserFindByName(w http.ResponseWriter, r *http.Request) {
	username, ok := r.Context().Value(usernameCtxKey).(string)
	if !ok {
		app.render(w, userFindTmpl, nil)
		return
	}

	u, err := app.Shimmie.GetUserByName(username)
	if err != nil {
		if err == sql.ErrNoRows {
			app.render(w, userFindTmpl, nil)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to retrieve user by name: %v", err), http.StatusInternalServerError)
		return
	}

	app.render(w, userFindTmpl, u)
}
