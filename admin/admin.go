//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"chichi/apis"

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

	// check the session cookie.
	var isLoggedIn bool
	cookie, err := r.Cookie("session")
	if err == nil {
		isLoggedIn = true
	}

	var accountID int
	var api *apis.AccountAPI
	var ws *apis.WorkspaceAPI
	if isLoggedIn {
		// get the account id
		accountID, err = strconv.Atoi(cookie.Value)
		if err != nil {
			log.Print(err)
		}

		// instantiate the account API
		api = admin.apis.AsAccount(accountID)

		// get the workspace
		ws = api.AsWorkspace(1) // TODO(marco)
	}

	// handle requests to login page.
	if rpath == "/" {
		if isLoggedIn {
			http.Redirect(w, r, "/admin/connectors", http.StatusTemporaryRedirect)
			return
		}
		if r.Method == "POST" {
			admin.login(w, r)
			return
		}
		http.ServeFile(w, r, "./admin/public/index.html")
		return
	}

	if strings.HasPrefix(rpath, "/src/") {
		admin.serveWithESBuild(w, r)
		return
	}

	if !isLoggedIn {
		http.Redirect(w, r, "/admin/", http.StatusTemporaryRedirect)
	}

	if strings.HasPrefix(rpath, "/add-data-source") {
		err = admin.serveAddDataSource(w, r, accountID)
		if err != nil {
			log.Printf("[error] %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}
	if strings.HasPrefix(rpath, "/oauth/authorize") {
		err = admin.serveAddOAuthDataSource(w, r, accountID)
		if err != nil {
			log.Printf("[error] %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if rpath == "/api/visualization" {
		admin.serveExecuteQuery(w, r)
		return
	}

	// Handle the "/list-users" endpoint.
	if strings.HasPrefix(rpath, "/list-users") {
		users, err := admin.apis.Users.Find()
		if err != nil {
			log.Printf("[error] cannot retrieve users: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(users)
		return
	}

	// Handle the "/user-schema-properties" endpoint.
	if strings.HasPrefix(rpath, "/user-schema-properties") {
		schema, err := ws.Schema("user")
		if err != nil {
			log.Printf("[error] cannot retrieve user schema: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var v struct {
			Properties map[string]any
		}
		err = json.Unmarshal([]byte(schema), &v)
		if err != nil {
			log.Printf("[error] cannot unmarshal user schema: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		props := make([]string, 0, len(v.Properties))
		for name := range v.Properties {
			props = append(props, name)
		}
		sort.Strings(props)
		_ = json.NewEncoder(w).Encode(props)
		return
	}

	// Handle the "/connectors-properties" endpoint.
	if strings.HasPrefix(rpath, "/connectors-properties") {
		var req struct {
			Connector int
		}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		properties, usedProperties, err := ws.DataSources.Properties(req.Connector)
		if err != nil {
			log.Printf("[error] cannot retrieve properties: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Properties":     properties,
			"UsedProperties": usedProperties,
		})
		return
	}

	// Handle the "/import-raw-user-data-from-connector" endpoint.
	if strings.HasPrefix(rpath, "/import-raw-user-data-from-connector") {
		var req struct {
			Connector   int
			ResetCursor bool
		}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		err = ws.DataSources.Import(req.Connector, req.ResetCursor)
		if err != nil {
			log.Printf("[error] %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		_, _ = fmt.Fprint(w, `{"status":"ok"}`)
		return
	}

	// Serve the transformation APIs.
	if strings.HasPrefix(rpath, "/transformations/") {
		rpath := rpath[len("/transformations"):]
		switch rpath {
		case "/get":
			var req struct {
				Connector int
			}
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			transf, err := ws.DataSources.TransformationFunc(req.Connector)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(transf)
		case "/update":
			var req struct {
				Connector      int
				Transformation string
			}
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			err = ws.DataSources.SetTransformationFunc(req.Connector, req.Transformation)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			_, _ = fmt.Fprint(w, `{"status":"ok"}`)
		}
		return
	}

	// Serve the schemas APIs.
	if strings.HasPrefix(rpath, "/schemas/") {
		rpath := rpath[len("/schemas"):]
		switch rpath {
		case "/get":
			var request struct {
				SchemaName string
			}
			err := json.NewDecoder(r.Body).Decode(&request)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			schema, err := ws.Schema(request.SchemaName)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(schema)
		case "/update":
			var request struct {
				SchemaName string
				Schema     string
			}
			err := json.NewDecoder(r.Body).Decode(&request)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			err = ws.SetSchema(request.SchemaName, request.Schema)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		default:
			http.NotFound(w, r)
		}
		return
	}

	if strings.HasPrefix(rpath, "/connectors/") {

		rpath := rpath[len("/connectors"):]

		switch rpath {
		case "/find":
			cns, err := admin.apis.Connectors()
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(cns)
			return
		case "/findInstalledConnectors":
			cns, err := ws.DataSources.List()
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(cns)
			return
		case "/get":
			var id int
			err := json.NewDecoder(r.Body).Decode(&id)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			cn, err := admin.apis.Connector(id)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ID": cn.ID, "Name": cn.Name, "LogoURL": cn.LogoURL, "OauthUrl": cn.OauthURL})
			return
		case "/delete":
			var ids []int
			err := json.NewDecoder(r.Body).Decode(&ids)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			err = ws.DataSources.Delete(ids[0])
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			return
		}
	}

	if strings.HasPrefix(rpath, "/properties/") {

		rpath := rpath[len("/properties"):]

		// Read the property ID from the headers.
		propertyID, _ := strconv.Atoi(r.Header.Get("X-Property"))
		if propertyID < 1 {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// TODO(Gianluca): check if the property belongs to the account.

		deprecatedProperty := api.DeprecatedProperty(propertyID)

		// Serve the Smart Event APIs.
		switch rpath {
		case "/smart-events.create":
			var event apis.SmartEventToCreate
			err := json.NewDecoder(r.Body).Decode(&event)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			id, err := deprecatedProperty.SmartEvents.Create(event)
			if err != nil {
				switch err.(type) {
				case apis.DomainNotAllowedError,
					apis.InvalidSmartEventError:
					http.Error(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
					return
				default:
					log.Printf("[error] %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			}
			_ = json.NewEncoder(w).Encode(id)
			return

		case "/smart-events.delete":
			var eventIDs []int
			err := json.NewDecoder(r.Body).Decode(&eventIDs)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			err = deprecatedProperty.SmartEvents.Delete(eventIDs)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			return

		case "/smart-events.find":
			smartEvents, err := deprecatedProperty.SmartEvents.Find()
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(smartEvents)
			return

		case "/smart-events.get":
			var id int
			err := json.NewDecoder(r.Body).Decode(&id)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			smartEvents, err := deprecatedProperty.SmartEvents.Get(id)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(smartEvents)
			return

		case "/smart-events.update":
			var req struct {
				ID         int
				SmartEvent apis.SmartEventToUpdate
			}
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			err = deprecatedProperty.SmartEvents.Update(req.ID, req.SmartEvent)
			if err != nil {
				switch err.(type) {
				case apis.DomainNotAllowedError,
					apis.InvalidSmartEventError:
					http.Error(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
					return
				default:
					log.Printf("[error] %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			}
			return

		}

	}

	http.ServeFile(w, r, "./admin/public/index.html")

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

	// TODO(Gianluca): fix this:
	columns, data, query, err := admin.apis.AsAccount(0).DeprecatedProperty(1).Visualization.ExecuteJSONQuery(context.TODO(), jsonQuery)
	if err != nil {
		switch err.(type) {
		case apis.InvalidJSONQueryError, apis.SmartEventNotFoundError:
			w.Header().Add("X-Error", err.Error())
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		default:
			log.Printf("[error] cannot execute query: %s", err)
			w.Header().Add("X-Error", err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
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

func (admin *admin) login(w http.ResponseWriter, r *http.Request) {
	loginData := struct {
		Email    string
		Password string
	}{}
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&loginData)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	enc := json.NewEncoder(w)
	accountID, err := admin.apis.Accounts.Authenticate(loginData.Email, loginData.Password)
	if err != nil {
		if err == apis.ErrAuthenticationFailed {
			enc.Encode([]any{0, "AuthenticationFailedError"})
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] cannot log account: %s", err)
		return
	}

	http.SetCookie(w, &http.Cookie{Name: "session", Value: strconv.Itoa(accountID), Path: "/"})
	w.WriteHeader(http.StatusOK)
	enc.Encode([]any{accountID, nil})
}

// serveAddDataSource serves a request to add a data source and responds with
// the data source identifier.
func (admin *admin) serveAddDataSource(w http.ResponseWriter, r *http.Request, accountID int) error {

	defer func() {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}()

	api := admin.apis.AsAccount(accountID)
	ws := api.AsWorkspace(1) // TODO(marco): what is the workspace?

	source := struct {
		Type      string
		Connector int
		Stream    int
	}{}
	err := json.NewDecoder(r.Body).Decode(&source)
	if err != nil {
		return err
	}

	var id int

	switch source.Type {
	case "App":
		id, err = ws.DataSources.AddApp(source.Connector, "", "")
	case "Database":
		id, err = ws.DataSources.AddDatabase(source.Connector)
	case "FileStream":
		id, err = ws.DataSources.AddFileStream(source.Connector, source.Stream)
	}
	if err != nil {
		return err
	}

	_ = json.NewEncoder(w).Encode(map[string]int{"id": id})

	return nil
}

// serveAddOAuthDataSource serves a request to add a data source authorized
// with OAuth and redirect to the confirmation page.
func (admin *admin) serveAddOAuthDataSource(w http.ResponseWriter, r *http.Request, accountID int) error {

	api := admin.apis.AsAccount(accountID)
	ws := api.AsWorkspace(1) // TODO(marco): what is the workspace?

	// Get the connector's identifier.
	cookie, err := r.Cookie("add-source")
	if err != nil {
		return errors.New("missing connector cookie")
	}
	defer func() {
		// Remove the "add-source" cookie.
		c := &http.Cookie{
			Name:     "add-source",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		}
		http.SetCookie(w, c)
	}()

	connectorID, err := strconv.Atoi(cookie.Value)
	if err != nil {
		return errors.New("invalid connector identifier")
	}

	// Get the OAuth code.
	oauthCode := r.URL.Query().Get("code")
	if oauthCode == "" {
		return errors.New("missing OAuth code from redirect URL")
	}

	connector, err := admin.apis.Connector(connectorID)
	if err != nil {
		return err
	}

	// Retrieve the refresh and access tokens.
	body := url.Values{}
	body.Set("grant_type", "authorization_code")
	body.Set("client_id", connector.ClientID)
	body.Set("client_secret", connector.ClientSecret)
	body.Set("redirect_uri", "https://localhost:9090/admin/oauth/authorize")
	body.Set("code", oauthCode)

	req, err := http.NewRequest("POST", connector.TokenEndpoint, strings.NewReader(body.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot retrieve the refresh and access tokens from connector %s: %s", connector.Name, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}()
	if resp.StatusCode != 200 {
		return fmt.Errorf("cannot retrieve the refresh and access tokens from connector %s: server responded with status %d", connector.Name, resp.StatusCode)
	}

	tokens := struct {
		Refresh string `json:"refresh_token"`
		Access  string `json:"access_token"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&tokens)
	if err != nil {
		return fmt.Errorf("cannot decode response from %s OAuth server: %s", connector.Name, err)
	}

	// Add the data source.
	_, err = ws.DataSources.AddApp(connectorID, tokens.Refresh, tokens.Access)
	if err != nil {
		return err
	}

	// Redirect to confirmation page.
	http.Redirect(w, r, "/admin/connectors/added/"+strconv.Itoa(connectorID), http.StatusTemporaryRedirect)

	return nil
}
