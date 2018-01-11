package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/kusubooru/shimmie"
)

func (app *App) serveSafe(w http.ResponseWriter, r *http.Request) {
	user, ok := shimmie.FromContextGetUser(r.Context())
	if !ok {
		http.Redirect(w, r, "/kanri", http.StatusFound)
		return
	}
	images, err := app.Shimmie.RatedImages.GetRatedImages(user.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		Images []shimmie.RatedImage
		Total  int
	}{
		images,
		len(images),
	}
	app.render(w, safeTmpl, data)
}

func (app *App) handleSafeRate(redirectURL string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST method.
		if r.Method != "POST" {
			http.Error(w, fmt.Sprintf("%v method not allowed", r.Method), http.StatusMethodNotAllowed)
			return
		}

		// Get user and user IP from context.
		user, ok := shimmie.FromContextGetUser(r.Context())
		if !ok {
			http.Redirect(w, r, "/kanri", http.StatusFound)
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
		err = app.Shimmie.Images.RateImage(imgID, imgRating)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Write the rating change to the shimmie log.
		err = app.Shimmie.Ratings.LogRating(imgID, imgRating, user.Name, userIP)
		if err != nil {
			log.Printf("Failed to log rating %q of image %d for username %q (%s) in score_log.",
				imgRating, imgID, user.Name, userIP)
		}

		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	})
}
