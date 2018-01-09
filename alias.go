package main

import (
	"fmt"
	"net/http"

	"github.com/kusubooru/shimmie"
)

func (app *App) serveAliasFind(w http.ResponseWriter, r *http.Request) {
	oldTag := r.FormValue("oldTag")
	newTag := r.FormValue("newTag")
	if oldTag == "" && newTag == "" {
		app.render(w, aliasFindTmpl, nil)
		return
	}

	alias, err := app.Shimmie.Aliases.FindAlias(oldTag, newTag)
	if err != nil {
		http.Error(w, fmt.Sprint("failed to find alias:", err), http.StatusInternalServerError)
		return
	}

	app.render(w, aliasFindTmpl, struct {
		Search shimmie.Alias
		Alias  []shimmie.Alias
	}{
		Search: shimmie.Alias{OldTag: oldTag, NewTag: newTag},
		Alias:  alias,
	})
}
