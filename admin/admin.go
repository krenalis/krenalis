//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"chichi/apis"
	"chichi/apis/errors"

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
	var account *apis.Account
	var workspace *apis.Workspace
	if isLoggedIn {
		// get the account id
		accountID, err = strconv.Atoi(cookie.Value)
		if err != nil {
			log.Print(err)
		}

		// instantiate the account API
		account, err = admin.apis.Accounts.As(accountID)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// get the workspace
		workspace, err = account.Workspaces.As(1) // TODO(marco)
		if err != nil {
			http.NotFound(w, r)
			return
		}

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

	if strings.HasPrefix(rpath, "/add-connection") {
		err = admin.serveAddConnection(w, r, accountID)
		if err != nil {
			log.Printf("[error] %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}
	if strings.HasPrefix(rpath, "/oauth/authorize") {
		err = admin.serveAddOAuthConnection(w, r, accountID)
		if err != nil {
			if err, ok := err.(errors.ResponseWriterTo); ok {
				_ = err.WriteTo(w)
				return
			}
			log.Printf("[error] %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if rpath == "/api/visualization" {
		admin.serveExecuteQuery(w, r)
		return
	}

	// Handle the "/user-schema-properties" endpoint.
	if strings.HasPrefix(rpath, "/user-schema") {
		dataType, _ := workspace.DataTypes.Get("user")
		w.Header().Add("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(dataType.Type)
		return
	}

	// Handle the "/group-schema-properties" endpoint.
	if strings.HasPrefix(rpath, "/group-schema-properties") {
		dataType, _ := workspace.DataTypes.Get("group")
		w.Header().Add("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(dataType.Type.PropertiesNames())
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
		err = workspace.Connections.Import(req.Connector, req.ResetCursor)
		if err != nil {
			log.Printf("[error] %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		return
	}

	// Handle the "/export" endpoint.
	if strings.HasPrefix(rpath, "/export") {
		var req struct {
			Connector int
		}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		err = workspace.Connections.Export(req.Connector)
		if err != nil {
			log.Printf("[error] %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		_, _ = fmt.Fprint(w, `{"status":"ok"}`)
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
			schema, _ := workspace.DataTypes.Get(request.SchemaName)
			w.Header().Add("Content-Type", "application/json")
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
			err = workspace.DataTypes.SetDefinition(request.SchemaName, request.Schema)
			if err != nil {
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		default:
			http.NotFound(w, r)
		}
		return
	}

	if strings.HasPrefix(rpath, "/connections/") {
		rpath := rpath[len("/connections"):]
		switch rpath {
		case "/find":
			connections := workspace.Connections.List()
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(connections)
			return
		case "/get":
			var id int
			err := json.NewDecoder(r.Body).Decode(&id)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			ds, err := workspace.Connections.Get(id)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ds)
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
			err = workspace.Connections.Delete(ids[0])
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			return
		case "/preview-query":
			defer r.Body.Close()
			var req struct {
				Connection int
				Query      string
				Limit      int
			}
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			columns, rows, err := workspace.Connections.Query(req.Connection, req.Query, req.Limit)
			if err != nil {
				if err, ok := err.(*errors.UnprocessableError); ok && err.Code == apis.QueryExecutionFailed {
					_ = json.NewEncoder(w).Encode(map[string]any{"Error": err.Err.Error()})
					return
				}
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"Columns": columns, "Rows": rows})
			return
		case "/set-users-query":
			defer r.Body.Close()
			var req struct {
				Connection int
				Query      string
			}
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			err = workspace.Connections.SetUsersQuery(req.Connection, req.Query)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			return
		case "/imports":
			defer r.Body.Close()
			var id int
			err := json.NewDecoder(r.Body).Decode(&id)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			imports, err := workspace.Connections.Imports(id)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(imports)
			return
		}
	}

	if strings.HasPrefix(rpath, "/connectors/") {
		rpath := rpath[len("/connectors"):]
		switch rpath {
		case "/find":
			connectors := admin.apis.Connectors.List()
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(connectors)
			return
		case "/get":
			var id int
			err := json.NewDecoder(r.Body).Decode(&id)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			conn, err := admin.apis.Connectors.Get(id)
			if err != nil {
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			var oAuthURL string
			if conn.OAuth != nil {
				oAuthURL = conn.OAuth.URL
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ID": conn.ID, "Name": conn.Name, "LogoURL": conn.LogoURL, "OAuthURL": oAuthURL})
			return
		case "/ui":
			var connection int
			err := json.NewDecoder(r.Body).Decode(&connection)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			form, err := workspace.Connections.ServeUI(connection, "load", nil)
			if err != nil {
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			_, err = w.Write(form)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			return
		case "/ui-event":
			var req struct {
				Connection int
				Event      string
				Values     json.RawMessage
			}
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			form, err := workspace.Connections.ServeUI(req.Connection, req.Event, req.Values)
			if err != nil {
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			_, err = w.Write(form)
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

		deprecatedProperty := account.DeprecatedProperty(propertyID)

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
			w.Header().Add("Content-Type", "application/json")
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
			w.Header().Add("Content-Type", "application/json")
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
			w.Header().Add("Content-Type", "application/json")
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
		JSX:               api.JSXAutomatic,
		LegalComments:     api.LegalCommentsEndOfFile,
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
	account, err := admin.apis.Accounts.As(0)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	columns, data, query, err := account.DeprecatedProperty(1).Visualization.ExecuteJSONQuery(context.TODO(), jsonQuery)
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
		if err, ok := err.(*errors.UnprocessableError); ok && err.Code == apis.AuthenticationFailed {
			enc.Encode([]any{0, "AuthenticationFailedError"})
			return
		}
		if err, ok := err.(errors.ResponseWriterTo); ok {
			_ = err.WriteTo(w)
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

// serveAddConnection serves a request to add a connection and responds with
// the connection identifier.
func (admin *admin) serveAddConnection(w http.ResponseWriter, r *http.Request, accountID int) error {

	defer func() {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}()

	account, err := admin.apis.Accounts.As(accountID)
	if err != nil {
		http.NotFound(w, r)
		return nil
	}
	workspace, err := account.Workspaces.As(1) // TODO(marco): what is the workspace?
	if err != nil {
		http.NotFound(w, r)
		return nil
	}

	connection := struct {
		Connector int
		Storage   int
		Role      string
		Host      string
	}{}
	err = json.NewDecoder(r.Body).Decode(&connection)
	if err != nil {
		return err
	}

	var role apis.ConnectionRole
	switch connection.Role {
	case "Source":
		role = apis.SourceRole
	case "Destination":
		role = apis.DestinationRole
	default:
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return nil
	}

	conn, err := admin.apis.Connectors.Get(connection.Connector)
	if err != nil {
		if err, ok := err.(errors.ResponseWriterTo); ok {
			_ = err.WriteTo(w)
			return nil
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] cannot get connector: %s", err)
		return nil
	}

	var opts apis.ConnectionOptions
	switch conn.Type {
	case apis.FileType:
		opts.Storage = connection.Storage
	case apis.WebsiteType:
		opts.WebsiteHost = connection.Host
	}
	id, err := workspace.Connections.Add(role, conn.ID, conn.Name, opts)
	if err != nil {
		return err
	}

	_ = json.NewEncoder(w).Encode(map[string]int{"id": id})

	return nil
}

// serveAddOAuthConnection serves a request to add a connection authorized with
// OAuth and redirect to the confirmation page.
func (admin *admin) serveAddOAuthConnection(w http.ResponseWriter, r *http.Request, accountID int) error {

	account, err := admin.apis.Accounts.As(accountID)
	if err != nil {
		return err
	}
	workspace, err := account.Workspaces.As(1) // TODO(marco): what is the workspace?
	if err != nil {
		return err
	}

	// Get the connector's identifier.
	idCookie, err := r.Cookie("add-connection")
	if err != nil {
		return errors.New("missing connector cookie")
	}

	// Get the role of the connection to add.
	roleCookie, err := r.Cookie("role")
	if err != nil {
		return errors.New("missing role cookie")
	}
	var role apis.ConnectionRole
	switch roleCookie.Value {
	case "Source":
		role = apis.SourceRole
	case "Destination":
		role = apis.DestinationRole
	default:
		return errors.New("unknown value in role cookie")
	}

	defer func() {
		// Remove the cookies.
		http.SetCookie(w, &http.Cookie{
			Name:     "add-connection",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "role",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
	}()

	connectorID, err := strconv.Atoi(idCookie.Value)
	if err != nil {
		return errors.New("invalid connector identifier")
	}

	// Get the OAuth code.
	oauthCode := r.URL.Query().Get("code")
	if oauthCode == "" {
		return errors.New("missing OAuth code from redirect URL")
	}

	connector, err := admin.apis.Connectors.Get(connectorID)
	if err != nil {
		return err
	}

	// Retrieve the refresh and access tokens.
	body := url.Values{}
	body.Set("grant_type", "authorization_code")
	body.Set("client_id", connector.OAuth.ClientID)
	body.Set("client_secret", connector.OAuth.ClientSecret)
	body.Set("redirect_uri", "https://localhost:9090/admin/oauth/authorize")
	body.Set("code", oauthCode)

	req, err := http.NewRequest("POST", connector.OAuth.TokenEndpoint, strings.NewReader(body.Encode()))
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
		// TODO(carlo): add Scope field and validate it
		AccessToken  string       `json:"access_token"`
		TokenType    string       `json:"token_type"` // TODO(carlo): validate the value
		ExpiresIn    *json.Number `json:"expires_in"` // TODO(carlo): validate the value
		RefreshToken string       `json:"refresh_token"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&tokens)
	if err != nil {
		return fmt.Errorf("cannot decode response from %s OAuth server: %s", connector.Name, err)
	}

	// TODO(carlo): compute the token type to use

	// Compute the access token expire time.
	expireIn := time.Now()
	if connector.OAuth.ForcedExpiresIn > 0 {
		expireIn = expireIn.Add(time.Duration(connector.OAuth.ForcedExpiresIn) * time.Second)
	} else if tokens.ExpiresIn != nil {
		seconds, _ := tokens.ExpiresIn.Int64()
		expireIn = expireIn.Add(time.Duration(seconds) * time.Second)
	} else if connector.OAuth.DefaultExpiresIn != 0 {
		expireIn = expireIn.Add(time.Duration(connector.OAuth.DefaultExpiresIn) * time.Second)
	}

	_, err = workspace.Connections.Add(role, connector.ID, connector.Name, apis.ConnectionOptions{
		OAuth: &apis.AddConnectionOAuthOptions{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			ExpiresIn:    expireIn,
		},
	})

	if err != nil {
		return err
	}

	// Redirect to confirmation page.
	http.Redirect(w, r, "/admin/connectors/added/"+strconv.Itoa(connectorID)+fmt.Sprintf("?role=%s", roleCookie.Value), http.StatusTemporaryRedirect)

	return nil
}
