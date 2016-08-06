package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kusubooru/shimmie"
	"github.com/kusubooru/shimmie/tags"
	"golang.org/x/net/context"
)

func (app *App) serveTagApproval(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// check for admin
	user, ok := ctx.Value("user").(*shimmie.User)
	if !ok || user.Admin != "Y" {
		http.Error(w, "You are not authorized to view this page.", http.StatusUnauthorized)
		return
	}

	// get owner username
	ownerUsername := r.FormValue("ownerUsername")
	ownerUsername = strings.TrimSpace(ownerUsername)
	if len(ownerUsername) == 0 {
		ownerUsername = "kusubooru"
	}

	ths, err := app.Shimmie.GetContributedTagHistory(ownerUsername)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	app.render(w, tagApprovalTmpl, ths)
}

func (app *App) serveTagHistory(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// check for admin
	user, ok := ctx.Value("user").(*shimmie.User)
	if !ok || user.Admin != "Y" {
		http.Error(w, "You are not authorized to view this page.", http.StatusUnauthorized)
		return
	}

	// get image id
	imageIDStr := r.FormValue("imageID")
	imageIDStr = strings.TrimSpace(imageIDStr)
	if len(imageIDStr) == 0 {
		app.render(w, tagHistoryTmpl, nil)
		return
	}

	// get tag history
	imageID, err := strconv.Atoi(imageIDStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad Image ID: %q, expecting integer: %v", imageIDStr, err), http.StatusBadRequest)
		return
	}

	ths, err := app.Shimmie.GetImageTagHistory(imageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	app.render(w, tagHistoryTmpl, ths)
}

func (app *App) serveTagsDiff(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	new := r.FormValue("new")
	old := r.FormValue("old")
	if len(new) == 0 && len(old) == 0 {
		app.render(w, tagsDiffTmpl, nil)
		return
	}
	newID, err := strconv.Atoi(new)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad tag history ID: %v, expecting integer: %v", new, err), http.StatusBadRequest)
		return
	}
	oldID, err := strconv.Atoi(old)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad tag history ID: %v, expecting integer: %v", old, err), http.StatusBadRequest)
		return
	}
	thNew, err := app.Shimmie.GetTagHistory(newID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	thOld, err := app.Shimmie.GetTagHistory(oldID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	removed, added := tags.DiffFields(thOld.Tags, thNew.Tags)

	app.render(w, tagsDiffTmpl, struct {
		Old     *shimmie.TagHistory
		New     *shimmie.TagHistory
		Removed []string
		Added   []string
	}{
		Old:     thOld,
		New:     thNew,
		Removed: removed,
		Added:   added,
	})
}
