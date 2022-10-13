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
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
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
	var api *apis.API
	if isLoggedIn {
		// get the account id
		accountID, err = strconv.Atoi(cookie.Value)
		if err != nil {
			log.Print(err)
		}

		// instantiate the account API
		api = admin.apis.API(accountID)
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
		admin.installConnector(w, r, accountID)
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
		propertyNames, err := admin.apis.Properties.UserSchemaProperties(accountID)
		if err != nil {
			log.Printf("[error] cannot retrieve properties: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(propertyNames)
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
		properties, err := api.DataSources.Properties(req.Connector)
		if err != nil {
			log.Printf("[error] cannot retrieve properties: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(properties)
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
		err = api.DataSources.Import(req.Connector, req.ResetCursor)
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
			transf, err := api.DataSources.TransformationFunc(req.Connector)
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
			err = api.DataSources.SetTransformationFunc(req.Connector, req.Transformation)
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
			schema, err := admin.apis.Schemas.Get(accountID, request.SchemaName)
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
			err = admin.apis.Schemas.Update(accountID, request.SchemaName, request.Schema)
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
			cns, err := api.DataSources.List()
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
			err = api.DataSources.Uninstall(ids[0])
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
	columns, data, query, err := admin.apis.API(0).DeprecatedProperty(1).Visualization.ExecuteJSONQuery(context.TODO(), jsonQuery)
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

// TODO(@Andrea): redirect to error screens with useful messages instead of
// sending generic internal server errors.
func (admin *admin) installConnector(w http.ResponseWriter, r *http.Request, accountID int) {

	api := admin.apis.API(1) // TODO(marco): what is the account?

	// get the ID of the connector.
	cookie, err := r.Cookie("install-connector")
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		log.Print("[error] cannot install connector: the request has not the cookie containing the connector ID")
		return
	}

	connectorID, err := strconv.Atoi(cookie.Value)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		log.Print("[error] cannot install connector: the connector ID contained in the cookie cannot be converted to int")
		return
	}

	// get the code from the query string.
	oauthCode := r.URL.Query().Get("code")
	if oauthCode == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		log.Printf("[error] cannot install connector %d: the redirect URI does not contain the oauth code", accountID)
		return
	}

	// retrieve the connector.
	connector, err := admin.apis.Connector(connectorID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] cannot install connector %d: %s", accountID, err)
		return
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", connector.ClientID)
	data.Set("client_secret", connector.ClientSecret)
	data.Set("redirect_uri", "https://localhost:9090/admin/oauth/authorize")
	data.Set("code", oauthCode)

	req, err := http.NewRequest(http.MethodPost, connector.TokenEndpoint, strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] cannot install connector %d: %s", connectorID, err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] cannot install connector %d: %s", connectorID, err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected status %d returned by connector %d while trying to get an access token via oauth code", resp.StatusCode, connectorID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] cannot install connector %d: %s", connectorID, err)
		return
	}

	respData := struct {
		Refresh_token string
	}{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&respData)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] cannot install connector %d: %s", connectorID, err)
		return
	}
	resp.Body.Close()

	err = api.DataSources.Install(connectorID, respData.Refresh_token)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] cannot install connector %d: %s", connectorID, err)
		return
	}

	// remove the "install-connector" cookie.
	c := &http.Cookie{
		Name:     "install-connector",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, c)

	// redirect to confirmation page.
	http.Redirect(w, r, "/admin/connectors/confirmation/"+strconv.Itoa(connectorID), http.StatusTemporaryRedirect)

	return
}

// connectorName returns the name of the connector with the given ID.
// If the ID does not correspond to any connector, returns "" and nil.
func (admin *admin) connectorName(id int) (string, error) {
	connector, err := admin.apis.Connector(id)
	if err != nil {
		return "", err
	}
	if connector == nil {
		return "", nil
	}
	return connector.Name, nil
}
