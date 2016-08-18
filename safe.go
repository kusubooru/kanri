package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/kusubooru/shimmie"
)

func (app *App) serveSafe(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(*shimmie.User)
	if !ok || user.Admin != "Y" {
		http.Error(w, "You are not authorized to view this page.", http.StatusUnauthorized)
		return
	}
	images, err := app.Shimmie.GetRatedImages(user.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	app.render(w, safeTmpl, images)
}

func (app *App) handleSafeRate(w http.ResponseWriter, r *http.Request) {
	// Only accept POST method.
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("%v method not allowed", r.Method), http.StatusMethodNotAllowed)
		return
	}

	// Get user and user IP from context.
	user, ok := r.Context().Value("user").(*shimmie.User)
	if !ok || user.Admin != "Y" {
		http.Error(w, "You are not authorized to view this page.", http.StatusUnauthorized)
		return
	}
	userIP := shimmie.GetOriginalIP(r)

	// Get image ID and rating from the HTTP request.
	id := r.PostFormValue("id")
	imgID, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid image ID: %v", err), http.StatusBadRequest)
		return
	}
	imgRating := r.PostFormValue("rating")

	// Store the new image rating.
	err = app.Shimmie.RateImage(imgID, imgRating)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the rating change to the shimmie log.
	err = app.Shimmie.LogRating(imgID, imgRating, user.Name, userIP)
	if err != nil {
		log.Printf("Failed to log rating %q of image %d for username %q (%s) in score_log.",
			imgRating, imgID, user.Name, userIP)
	}

	// Serve again the safe approval template.
	app.serveSafe(w, r)
}
