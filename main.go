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
	"strings"

	"github.com/kusubooru/shimmie"
	"github.com/kusubooru/shimmie/store"
)

//go:generate go run generate/templates.go

const (
	description = `Usage: kanri [options]
  Management tools for Shimmie2.
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
	var (
		theVersion  = "devel"
		httpAddr    = flag.String("http", "localhost:8080", "HTTP listen address")
		dbDriver    = flag.String("dbdriver", "mysql", "database driver")
		dbConfig    = flag.String("dbconfig", "", "username:password@(host:port)/database?parseTime=true")
		loginURL    = flag.String("loginurl", "/kanri/login", "login URL path to redirect to")
		imagePath   = flag.String("imagepath", "", "path where images are stored")
		thumbPath   = flag.String("thumbpath", "", "path where image thumbnails are stored")
		showVersion = flag.Bool("v", false, "print program version")
		certFile    = flag.String("tlscert", "", "TLS public key in PEM format.  Must be used together with -tlskey")
		keyFile     = flag.String("tlskey", "", "TLS private key in PEM format.  Must be used together with -tlscert")
		// Set after flag parsing based on certFile & keyFile.
		useTLS bool
		// Set after flag parsing; true if "version" is first argument.
		versionArg bool
	)
	flag.Usage = usage
	flag.Parse()
	useTLS = *certFile != "" && *keyFile != ""
	versionArg = len(os.Args) > 1 && os.Args[1] == "version"

	if *showVersion || versionArg {
		fmt.Printf("%s %s (runtime: %s)\n", os.Args[0], theVersion, runtime.Version())
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

	http.Handle("/kanri", shim.Auth(app.serveIndex, *loginURL))
	http.Handle("/kanri/safe", shim.Auth(mustAdmin(app.serveSafe), *loginURL))
	http.Handle("/kanri/safe/rate", shim.Auth(mustAdmin(app.handleSafeRate), *loginURL))
	http.Handle("/kanri/_image/", shim.Auth(app.serveImage, *loginURL))
	http.Handle("/kanri/_thumb/", shim.Auth(app.serveThumb, *loginURL))
	http.Handle("/kanri/login", http.HandlerFunc(app.serveLogin))
	http.Handle("/kanri/login/submit", http.HandlerFunc(app.handleLogin))
	http.Handle("/kanri/logout", http.HandlerFunc(handleLogout))
	http.Handle("/kanri/tags/history", shim.Auth(mustAdmin(app.serveTagHistory), *loginURL))
	http.Handle("/kanri/tags/history/diff", shim.Auth(mustAdmin(app.handleTagHistoryDiff), *loginURL))
	http.Handle("/kanri/tags/diff", shim.Auth(app.serveTagsDiff, *loginURL))
	http.Handle("/kanri/tags/approval", shim.Auth(mustAdmin(app.serveTagApproval), *loginURL))
	http.Handle("/kanri/user/find", shim.Auth(mustAdmin(app.serveUserFind), *loginURL))

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

type App struct {
	Shimmie *shimmie.Shimmie
	Common  *shimmie.Common
}

func (app *App) serveIndex(w http.ResponseWriter, r *http.Request) {
	app.render(w, indexTmpl, nil)
}

func (app *App) serveImage(w http.ResponseWriter, r *http.Request) {
	app.serveImageFile(w, r, app.Shimmie.ImagePath)
}

func (app *App) serveThumb(w http.ResponseWriter, r *http.Request) {
	app.serveImageFile(w, r, app.Shimmie.ThumbPath)
}

func (app *App) serveImageFile(w http.ResponseWriter, r *http.Request, path string) {
	hash := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]

	err := app.Shimmie.WriteImageFile(w, path, hash)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not write image file: %v", err), http.StatusInternalServerError)
		return
	}
}

func (app *App) serveLogin(w http.ResponseWriter, r *http.Request) {
	app.render(w, loginTmpl, nil)
}

func (app *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	// only accept POST method
	if r.Method != "POST" {
		http.Error(w, fmt.Sprintf("%v method not allowed", r.Method), http.StatusMethodNotAllowed)
		return
	}
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")
	user, err := app.Shimmie.GetUserByName(username)
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
	if r.Method != "POST" {
		http.Error(w, "Use POST to logout.", http.StatusMethodNotAllowed)
		return
	}
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

// mustAdmin checks context to see if user is admin and sends error
// Unauthorized ifthey are not.
func mustAdmin(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user and user IP from context.
		user, ok := shimmie.FromContextGetUser(r.Context())
		if !ok || user.Admin != "Y" {
			http.Error(w, "You are not authorized to view this page.", http.StatusUnauthorized)
			return
		}
		fn.ServeHTTP(w, r)
	}
}
