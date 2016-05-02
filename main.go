package main

import (
	"crypto/md5"
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/net/context"

	"github.com/kusubooru/kanri/version"
	"github.com/kusubooru/shimmie"
	"github.com/kusubooru/shimmie/store"
)

//go:generate go run generate/templates.go

var (
	httpAddr    = flag.String("http", "localhost:8080", "HTTP listen address")
	dbDriver    = flag.String("dbdriver", "mysql", "database driver")
	dbConfig    = flag.String("dbconfig", "", "username:password@(host:port)/database?parseTime=true")
	loginURL    = flag.String("loginurl", "/kanri/login", "login URL path to redirect to")
	imagePath   = flag.String("imagepath", "images", "path where images are stored")
	thumbPath   = flag.String("thumbpath", "thumbs", "path where image thumbnails are stored")
	showVersion = flag.Bool("v", false, "print program version")
	certFile    = flag.String("tlscert", "", "TLS public key in PEM format.  Must be used together with -tlskey")
	keyFile     = flag.String("tlskey", "", "TLS private key in PEM format.  Must be used together with -tlscert")
	// Set after flag parsing based on certFile & keyFile.
	useTLS     bool
	versionArg bool
)

const (
	description = `Usage: kanri [options]
  Management tools for shimmie.
Options:
`
)

func usage() {
	fmt.Fprintf(os.Stderr, description)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n")
}

var fns = template.FuncMap{
	"join": strings.Join,
}

func main() {
	flag.Usage = usage
	flag.Parse()
	useTLS = *certFile != "" && *keyFile != ""
	versionArg = len(os.Args) > 1 && os.Args[1] == "version"

	if *showVersion || versionArg {
		fmt.Printf("kanri version %v %v/%v\n", version.Core, runtime.GOOS, runtime.GOARCH)
		return
	}

	// open store with new database connection and create new Shimmie
	shim := shimmie.New(*imagePath, *thumbPath, store.Open(*dbDriver, *dbConfig))

	// get common conf
	common, cerr := shim.Store.GetCommon()
	if cerr != nil {
		log.Fatalln("could not get common conf:", cerr)
	}

	// app with Shimmie and common conf
	app := App{Shimmie: shim, Common: common}

	// new context
	ctx := context.Background()

	http.Handle("/kanri", shim.Auth(ctx, app.serveIndex, *loginURL))
	http.Handle("/kanri/safe", shim.Auth(ctx, app.serveSafe, *loginURL))
	http.Handle("/kanri/safe/rate", shim.Auth(ctx, app.handleSafeRate, *loginURL))
	http.Handle("/kanri/_image/", shim.Auth(ctx, app.serveImage, *loginURL))
	http.Handle("/kanri/_thumb/", shim.Auth(ctx, app.serveThumb, *loginURL))
	http.Handle("/kanri/login", newHandler(ctx, app.serveLogin))
	http.Handle("/kanri/login/submit", newHandler(ctx, app.handleLogin))
	http.Handle("/kanri/logout", http.HandlerFunc(handleLogout))

	if useTLS {
		if err := http.ListenAndServeTLS(*httpAddr, *certFile, *keyFile, nil); err != nil {
			log.Fatalf("Could not start listening (TLS) on %v: %v", *httpAddr, err)
		}
	} else {
		if err := http.ListenAndServe(*httpAddr, nil); err != nil {
			log.Fatalf("Could not start listening on %v: %v", *httpAddr, err)
		}
	}
}

type ctxHandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

func newHandler(ctx context.Context, fn ctxHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(ctx, w, r)
	}
}

type App struct {
	Shimmie *shimmie.Shimmie
	Common  *shimmie.Common
}

func (app *App) serveIndex(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	app.render(w, indexTmpl, nil)
}

func (app *App) serveSafe(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	user, ok := ctx.Value("user").(*shimmie.User)
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

func (app *App) handleSafeRate(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Only accept POST method.
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("%v method not allowed", r.Method), http.StatusMethodNotAllowed)
		return
	}

	// Get user and user IP from context.
	user, ok := ctx.Value("user").(*shimmie.User)
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
	app.serveSafe(ctx, w, r)
}

func (app *App) serveImage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	app.serveImageFile(ctx, w, r, app.Shimmie.ImagePath)
}

func (app *App) serveThumb(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	app.serveImageFile(ctx, w, r, app.Shimmie.ThumbPath)
}

func (app *App) serveImageFile(ctx context.Context, w http.ResponseWriter, r *http.Request, path string) {
	hash := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]

	err := app.Shimmie.WriteImageFile(w, path, hash)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not write image file: %v", err), http.StatusInternalServerError)
		return
	}
}

func (app *App) serveLogin(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	app.render(w, loginTmpl, nil)
}

func (app *App) handleLogin(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// only accept POST method
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("%v method not allowed", r.Method), http.StatusMethodNotAllowed)
		return
	}
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")
	user, err := app.Shimmie.GetUser(username)
	if err != nil {
		if err == sql.ErrNoRows {
			app.render(w, loginTmpl, "User does not exist.")
			return
		}
		msg := fmt.Sprintf("get user %q failed: %v", username, err.Error())
		log.Print(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	hash := md5.Sum([]byte(username + password))
	passwordHash := fmt.Sprintf("%x", hash)
	if user.Pass != passwordHash {
		app.render(w, loginTmpl, "Username and password do not match.")
		return
	}
	addr := strings.Split(r.RemoteAddr, ":")[0]
	cookieValue := shimmie.CookieValue(passwordHash, addr)
	shimmie.SetCookie(w, "shm_user", username)
	shimmie.SetCookie(w, "shm_session", cookieValue)
	http.Redirect(w, r, "/kanri", http.StatusFound)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	shimmie.SetCookie(w, "shm_user", "")
	shimmie.SetCookie(w, "shm_session", "")
	http.Redirect(w, r, "/kanri", http.StatusFound)
}

func render(w http.ResponseWriter, t *template.Template, data interface{}) {
	if err := t.Execute(w, data); err != nil {
		msg := fmt.Sprintln("could not render template:", err)
		log.Print(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
}

func (app *App) render(w http.ResponseWriter, t *template.Template, data interface{}) {
	render(w, t, struct {
		Data   interface{}
		Common *shimmie.Common
	}{
		Data:   data,
		Common: app.Common,
	})
}
