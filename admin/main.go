package admin

import (
	"chichi/apis"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

type admin struct {
	apis *apis.APIs
}

func New(apis *apis.APIs) *admin {
	return &admin{apis: apis}
}

func (admin *admin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rpath := r.URL.Path[6:]

	if rpath == "/login/" {
		return
		// TODO(@Andrea): implement login
	}

	cookie, err := r.Cookie("session")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		log.Print(err)
	}

	// get the customer id
	customerID, err := strconv.Atoi(cookie.Value)
	if err != nil {
		log.Print(err)
	}

	// istantiate the customer api
	API := admin.apis.API(customerID)
	_ = API

	if strings.HasPrefix(rpath, "/public/") {
		http.ServeFile(w, r, "./admin/public/index.html")
	}

	if strings.HasPrefix(rpath, "/src/") {
		admin.serveWithESBuild(w, r)
	}

	if rpath == "/api/visualization" {
		admin.serveExecuteQuery(w, r)
	}

}

func (admin *admin) serveWithESBuild(w http.ResponseWriter, r *http.Request) {
	file, err := filepath.Abs("admin/src/index.js")
	if err != nil {
		panic(err)
	}
	result := api.Build(api.BuildOptions{
		Bundle:            true,
		EntryPoints:       []string{file},
		Format:            api.FormatESModule,
		JSXMode:           api.JSXModeAutomatic,
		Loader:            map[string]api.Loader{".js": api.LoaderJSX},
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		MinifyWhitespace:  true,
		Outdir:            "out",
		Target:            api.ES2018,
		TreeShaking:       api.TreeShakingTrue,
		Write:             false,
	})

	// Handle errors and warnings.
	if result.Errors != nil {
		errorMessages := &strings.Builder{}
		for _, msg := range result.Errors {
			log.Printf("[error] ESBuild error: %v", msg)
			errorMessages.WriteString(fmt.Sprint(msg))
		}
		log.Printf("[error] errors while executing ESbuild, cannot serve %q", r.URL.Path)
		http.Error(w, errorMessages.String(), http.StatusInternalServerError)
		return
	}
	if result.Warnings != nil {
		for _, msg := range result.Warnings {
			log.Printf("[warning] ESBuild warning: %v", msg)
		}
	}

	base := path.Base(r.URL.Path)
	for _, out := range result.OutputFiles {
		if strings.HasSuffix(out.Path, base) {
			switch filepath.Ext(base) {
			case ".js":
				w.Header().Add("Content-Type", "text/javascript")
			case ".css":
				w.Header().Add("Content-Type", "text/css")
			default:
				log.Printf("[error] cannot determine Content-Type for %q", base)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			w.Write(out.Contents)
			return
		}
	}
	http.NotFound(w, r)
}

func (admin *admin) serveExecuteQuery(w http.ResponseWriter, r *http.Request) {
	// Parse the request.
	var jsonQuery apis.JSONQuery
	err := json.NewDecoder(r.Body).Decode(&jsonQuery)
	if err != nil {
		w.Header().Add("X-Error", err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	columns, data, query, err := admin.apis.API(0).Properties.Visualization.ExecuteQuery(context.TODO(), jsonQuery)
	if err != nil {
		log.Printf("[error] cannot execute query: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	// Send the results to the client.
	w.Header().Add("Content-Type", "application/json")
	var response struct {
		Columns []string
		Data    [][]any
		Query   string
	}
	response.Columns = columns
	response.Data = data
	response.Query = query
	_ = json.NewEncoder(w).Encode(response)
}
