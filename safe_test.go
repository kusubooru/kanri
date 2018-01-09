package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kusubooru/kanri/internal/mock"
	"github.com/kusubooru/shimmie"
)

//type _shimmie struct{}
//
//func (s _shimmie) LogRating(imgID int, rating, username, userIP string) error { return nil }
//func (s _shimmie) RateImage(id int, rating string) error                      { return nil }

func TestApp_handleSafeRate(t *testing.T) {
	//shim := _shimmie{}
	shim := &mock.Shimmie{}
	shim.LogRatingFn = func(imgID int, rating, username, userIP string) error { return nil }
	shim.RateImageFn = func(id int, rating string) error { return nil }

	app := App{
		Shimmie: Shimmie{
			Images:  shim,
			Ratings: shim,
		},
	}

	v := url.Values{}
	v.Set("id", "1")
	req := httptest.NewRequest("POST", "/foo", strings.NewReader(v.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(shimmie.NewContextWithUser(req.Context(), &shimmie.User{}))
	w := httptest.NewRecorder()
	handler := app.handleSafeRate("/bar")
	handler(w, req)

	resp := w.Result()

	if got, want := resp.StatusCode, http.StatusSeeOther; got != want {
		t.Errorf("StatusCode = %d, want %d", got, want)
	}
	if got, want := resp.Header.Get("Location"), "/bar"; got != want {
		t.Errorf("Location = %s, want %s", got, want)
	}
}
