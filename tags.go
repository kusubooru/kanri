package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kusubooru/kanri/tags"
	"github.com/kusubooru/shimmie"
)

func (app *App) serveTagApproval(w http.ResponseWriter, r *http.Request) {
	// get owner username
	ownerUsername := r.FormValue("ownerUsername")
	ownerUsername = strings.TrimSpace(ownerUsername)
	if len(ownerUsername) == 0 {
		ownerUsername = "kusubooru"
	}

	ths, err := app.Shimmie.ContributedTagHistories.GetContributedTagHistory(ownerUsername)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	app.render(w, tagApprovalTmpl, ths)
}

func (app *App) serveTagHistory(w http.ResponseWriter, r *http.Request) {
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

	ths, err := app.Shimmie.ImageTagHistories.GetImageTagHistory(imageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	app.render(w, tagHistoryTmpl, ths)
}

func (app *App) serveTagsDiff(w http.ResponseWriter, r *http.Request) {
	old := r.PostFormValue("old")
	new := r.PostFormValue("new")
	removed, added := tags.DiffFields(old, new)

	data := struct {
		Old     string
		New     string
		Removed []string
		Added   []string
	}{old, new, removed, added}

	app.render(w, tagsDiffTmpl, data)
}

func (app *App) handleTagHistoryDiff(w http.ResponseWriter, r *http.Request) {
	new := r.FormValue("new")
	old := r.FormValue("old")
	if len(new) == 0 && len(old) == 0 {
		app.render(w, tagHistoryDiffTmpl, nil)
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
	thNew, err := app.Shimmie.TagHistories.GetTagHistory(newID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	thOld, err := app.Shimmie.TagHistories.GetTagHistory(oldID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	removed, added := tags.DiffFields(thOld.Tags, thNew.Tags)

	app.render(w, tagHistoryDiffTmpl, struct {
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

func (app *App) serveTagsScan(w http.ResponseWriter, r *http.Request) {
	input := r.PostFormValue("input")
	tgs := tags.Scan(input)

	app.render(w, tagsScanTmpl, struct {
		Input string
		Tags  string
	}{
		Input: input,
		Tags:  strings.Join(tgs, " "),
	})
}
