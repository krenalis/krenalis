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
	"log"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"

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
		account, err := admin.apis.Account(accountID)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// get the workspace
		workspace, err = account.Workspace(1) // TODO(marco)
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

	if strings.HasPrefix(rpath, "/oauth/authorize") {
		oauthCode := r.URL.Query().Get("code")
		if oauthCode == "" {
			log.Printf("[error] %v", errors.New("missing OAuth code from redirect URL"))
			http.Redirect(w, r, "/admin/oauth/error", http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w, r, "/admin/oauth"+fmt.Sprintf("?oauthCode=%s", url.QueryEscape(oauthCode)), http.StatusTemporaryRedirect)
		return
	}

	// Handle the "/predefined-mappings" endpoint.
	if strings.HasPrefix(rpath, "/predefined-mappings") {
		funcs := make([]map[string]any, len(apis.PredefinedMappingFuncs))
		for i, f := range apis.PredefinedMappingFuncs {
			funcs[i] = map[string]any{
				"ID":          f.ID,
				"Name":        f.Name,
				"Description": f.Description,
				"Icon":        f.Icon,
				"In":          f.In,
				"Out":         f.Out,
			}
		}
		w.Header().Add("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(funcs)
		return
	}

	if rpath == "/api/visualization" {
		admin.serveExecuteQuery(w, r)
		return
	}

	// Handle the "/user-schema-properties" endpoint.
	if strings.HasPrefix(rpath, "/user-schema") {
		schema := workspace.Schema("users")
		w.Header().Add("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(schema)
		return
	}

	// Handle the "/group-schema-properties" endpoint.
	if strings.HasPrefix(rpath, "/group-schema-properties") {
		schema := workspace.Schema("groups")
		w.Header().Add("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(schema.PropertiesNames())
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
		connection, err := workspace.Connection(req.Connector)
		if err != nil {
			log.Printf("[error] %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		err = connection.Import(req.ResetCursor)
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
		connection, err := workspace.Connection(req.Connector)
		if err != nil {
			log.Printf("[error] %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		err = connection.Export()
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
			schema := workspace.Schema(request.SchemaName)
			w.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(schema)
		default:
			http.NotFound(w, r)
		}
		return
	}

	if strings.HasPrefix(rpath, "/connections/") {
		rpath := rpath[len("/connections"):]
		switch rpath {
		case "/find":
			connections := workspace.Connections()
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
			connection, err := workspace.Connection(id)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(connection)
			return
		case "/delete":
			var ids []int
			err := json.NewDecoder(r.Body).Decode(&ids)
			if err != nil || len(ids) == 0 {
				log.Printf("[error] %v", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			connection, err := workspace.Connection(ids[0])
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			err = connection.Delete()
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
			connection, err := workspace.Connection(req.Connection)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			columns, rows, err := connection.Query(req.Query, req.Limit)
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
			connection, err := workspace.Connection(req.Connection)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			err = connection.SetUsersQuery(req.Query)
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
			connection, err := workspace.Connection(id)
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			imports, err := connection.Imports()
			if err != nil {
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(imports)
			return
		case "/ui":
			var id int
			err := json.NewDecoder(r.Body).Decode(&id)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			connection, err := workspace.Connection(id)
			if err != nil {
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			form, err := connection.ServeUI("load", nil)
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
			connection, err := workspace.Connection(req.Connection)
			if err != nil {
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			form, err := connection.ServeUI(req.Event, req.Values)
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

	if strings.HasPrefix(rpath, "/connectors/") {
		rpath := rpath[len("/connectors"):]
		switch rpath {
		case "/find":
			connectors := admin.apis.Connectors()
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
			conn, err := admin.apis.Connector(id)
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
			_ = json.NewEncoder(w).Encode(map[string]any{"ID": conn.ID, "Name": conn.Name, "LogoURL": conn.LogoURL, "OAuthURL": oAuthURL, "Type": conn.Type, "HasSettings": conn.HasSettings})
			return
		case "/ui":
			var req struct {
				Connector  int
				Role       string
				OAuthToken string
			}
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			connector, err := admin.apis.Connector(req.Connector)
			if err != nil {
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var role apis.ConnectionRole
			switch req.Role {
			case "Source":
				role = apis.SourceRole
			case "Destination":
				role = apis.DestinationRole
			default:
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			form, err := connector.ServeUI("load", nil, role, req.OAuthToken)
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
				Connector  int
				Event      string
				Values     json.RawMessage
				Role       string
				OAuthToken string
			}
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			connector, err := admin.apis.Connector(req.Connector)
			if err != nil {
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			var role apis.ConnectionRole
			switch req.Role {
			case "Source":
				role = apis.SourceRole
			case "Destination":
				role = apis.DestinationRole
			default:
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			form, err := connector.ServeUI(req.Event, req.Values, role, req.OAuthToken)
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
	account, err := admin.apis.Account(0)
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
	accountID, err := admin.apis.AuthenticateAccount(loginData.Email, loginData.Password)
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
