// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package workos

import (
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"

	"golang.org/x/text/unicode/norm"
)

// maxPayloadSize is the maximum size in bytes for a webhook or action payload.
const maxPayloadSize = 64 * 1024

// ServeHTTP serves action and webhook requests.
func (wo *WorkOS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/actions/user-registration":
		wo.serveAction(w, r)
		return
	case "/webhook":
		wo.serveWebhook(w, r)
		return
	}
	http.Error(w, "Not Found", http.StatusNotFound)
}

// ServeLogin authenticates a member through WorkOS. It verifies the WorkOS
// access token, looks up the member by WorkOS user ID, provisions the member
// if they are not already registered in Krenalis, and returns the member ID.
func (wo *WorkOS) ServeLogin(r *http.Request) (string, error) {

	var body struct {
		AccessToken string `json:"accessToken"`
	}
	err := json.Decode(r.Body, &body)
	if err != nil || body.AccessToken == "" {
		return "", errors.BadRequest("")
	}

	workosUser, err := wo.authenticate(r.Context(), body.AccessToken)
	if err != nil {
		if errors.Is(err, errAuthenticationFailed) {
			return "", errors.Unauthorized("invalid WorkOS token")
		}
		return "", err
	}

	email := strings.TrimSpace(norm.NFC.String(workosUser.Email))
	firstName := strings.TrimSpace(norm.NFC.String(workosUser.FirstName))
	lastName := strings.TrimSpace(norm.NFC.String(workosUser.LastName))

	org, err := wo.core.Organization(workosUser.OrganizationExternalID)
	if err != nil {
		if _, ok := err.(*errors.NotFoundError); ok {
			slog.Error("WorkOS login rejected: organization does not exist",
				"workos_user", workosUser.ID,
				"organization", workosUser.OrganizationExternalID,
			)
			return "", errors.Unauthorized("invalid organization ID in WorkOS token")
		}
		return "", err
	}
	if !org.Enabled {
		return "", errors.Unprocessable(core.OrganizationDisabled, "organization %s is disabled", org.ID)
	}

	member, err := org.MemberByWorkOSID(r.Context(), workosUser.ID)
	if err != nil {
		if _, ok := err.(*errors.NotFoundError); !ok {
			return "", err
		}
		name := firstName + " " + lastName
		member, err = org.AddMember(r.Context(), core.MemberToSet{Name: name, Email: email, WorkOSUserID: workosUser.ID})
		if e, ok := err.(*errors.UnprocessableError); ok && (e.Code == core.MemberEmailExists || e.Code == core.MemberWorkOSUserIDExists) {
			member, err = org.MemberByWorkOSID(r.Context(), workosUser.ID)
		}
		if err != nil {
			return "", err
		}
	}

	return member, nil
}

// serveAction handles the user registration action. It verifies the request
// signature and rejects the registration if the user's email address does not
// match the one on the WorkOS invitation.
func (wo *WorkOS) serveAction(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadSize)
	rawBody, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			_ = errors.BadRequest("request body too large").WriteTo(w)
			return
		}
		_ = errors.BadRequest("failed to read request body").WriteTo(w)
		return
	}

	sigHeader := r.Header.Get("WorkOS-Signature")
	if sigHeader == "" {
		_ = errors.Unauthorized("WorkOS action is missing the signature header").WriteTo(w)
		return
	}
	if err := wo.verifyActionSignature(rawBody, sigHeader); err != nil {
		_ = errors.Unauthorized("invalid WorkOS action signature").WriteTo(w)
		return
	}

	var action struct {
		ID       string `json:"id"`
		Object   string `json:"object"`
		UserData struct {
			Email string `json:"email"`
		} `json:"user_data"`
		Invitation *struct {
			Email string `json:"email"`
		} `json:"invitation"`
	}
	if err := json.Unmarshal(rawBody, &action); err != nil {
		_ = errors.BadRequest("invalid action payload").WriteTo(w)
		return
	}

	slog.Info("WorkOS action received", "id", action.ID, "object", action.Object)

	verdict, message := "Deny", "Registration is by invitation only."

	if action.Invitation != nil {
		userEmail := strings.TrimSpace(norm.NFC.String(action.UserData.Email))
		invitationEmail := strings.TrimSpace(norm.NFC.String(action.Invitation.Email))
		if strings.EqualFold(userEmail, invitationEmail) {
			verdict, message = "Allow", ""
			slog.Info("WorkOS action: registration allowed", "id", action.ID)
		} else {
			message = "You must register with the email address you were invited with."
			slog.Info("WorkOS action: registration denied: email mismatch", "id", action.ID)
		}
	} else {
		slog.Info("WorkOS action: registration denied: no invitation", "id", action.ID)
	}

	responseJSON, err := wo.buildActionResponse(verdict, message)
	if err != nil {
		slog.Error("WorkOS action error: failed to build response", "id", action.ID, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(responseJSON)
}

// serveWebhook handles incoming webhook events.
func (wo *WorkOS) serveWebhook(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadSize)
	rawBody, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			_ = errors.BadRequest("request body too large").WriteTo(w)
			return
		}
		_ = errors.BadRequest("failed to read request body").WriteTo(w)
		return
	}

	sigHeader := r.Header.Get("WorkOS-Signature")
	if sigHeader == "" {
		_ = errors.Unauthorized("WorkOS webhook is missing the signature header").WriteTo(w)
		return
	}
	err = wo.verifyWebhookSignature(rawBody, sigHeader)
	if err != nil {
		_ = errors.Unauthorized("invalid WorkOS webhook signature").WriteTo(w)
		return
	}

	var event struct {
		ID    string `json:"id"`
		Event string `json:"event"`
		Data  struct {
			ID             string  `json:"id"`
			Email          string  `json:"email"`
			FirstName      string  `json:"first_name"`
			LastName       string  `json:"last_name"`
			Name           string  `json:"name"`
			ExternalID     *string `json:"external_id"`
			UserID         string  `json:"user_id"`
			OrganizationID string  `json:"organization_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rawBody, &event); err != nil {
		_ = errors.BadRequest("invalid webhook payload").WriteTo(w)
		return
	}

	slog.Info("WorkOS webhook received", "id", event.ID, "event", event.Event)

	switch event.Event {
	case "user.updated":
		email := strings.TrimSpace(norm.NFC.String(event.Data.Email))
		firstName := strings.TrimSpace(norm.NFC.String(event.Data.FirstName))
		lastName := strings.TrimSpace(norm.NFC.String(event.Data.LastName))
		name := firstName + " " + lastName
		if event.Data.ID == "" || email == "" {
			slog.Info("WorkOS webhook: skipping user.updated: missing user ID or email", "id", event.ID)
			return
		}
		if runes := []rune(name); len(runes) > 255 {
			name = string(runes[:255])
		}
		if err := wo.core.UpdateMembersByWorkOSID(r.Context(), event.Data.ID, name, email); err != nil {
			if e, ok := err.(*errors.UnprocessableError); ok && e.Code == core.MemberEmailExists {
				// Email already in use, skip the update without returning
				// errors to prevent webhook retries.
				slog.Error("WorkOS webhook error: cannot update member's email because the new email already exists", "id", event.ID, "workos_user", event.Data.ID)
				return
			}
			slog.Error("WorkOS webhook error: failed to update member", "id", event.ID, "workos_user", event.Data.ID, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		slog.Info("WorkOS webhook: member updated", "id", event.ID, "workos_user", event.Data.ID)
	case "user.deleted":
		if event.Data.ID == "" {
			slog.Info("WorkOS webhook: skipping user.deleted: missing user ID", "id", event.ID)
			return
		}
		if err := wo.core.DeleteMembersByWorkOSID(r.Context(), event.Data.ID); err != nil {
			slog.Error("WorkOS webhook error: failed to delete member", "id", event.ID, "workos_user", event.Data.ID, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		slog.Info("WorkOS webhook: member deleted", "id", event.ID, "workos_user", event.Data.ID)
	case "organization.updated":
		if event.Data.ExternalID == nil || *event.Data.ExternalID == "" {
			slog.Info("WorkOS webhook: skipping organization.updated: missing external ID", "id", event.ID)
			return
		}
		orgID := *event.Data.ExternalID
		orgName := strings.TrimSpace(norm.NFC.String(event.Data.Name))
		if orgName == "" {
			slog.Info("WorkOS webhook: skipping organization.updated: missing organization name", "id", event.ID, "organization", orgID)
			return
		}
		if runes := []rune(orgName); len(runes) > 255 {
			orgName = string(runes[:255])
		}
		org, err := wo.core.Organization(orgID)
		if err != nil {
			if _, ok := err.(*errors.NotFoundError); ok {
				slog.Info("WorkOS webhook: skipping organization.updated: organization not found", "id", event.ID, "organization", orgID)
				return
			}
			slog.Error("WorkOS webhook error: failed to get organization", "id", event.ID, "organization", orgID, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if err := org.Update(r.Context(), orgName, nil); err != nil {
			slog.Error("WorkOS webhook error: failed to update organization", "id", event.ID, "organization", orgID, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		slog.Info("WorkOS webhook: organization updated", "id", event.ID, "organization", orgID)
	case "organization_membership.created":
		if event.Data.UserID == "" || event.Data.OrganizationID == "" {
			slog.Info("WorkOS webhook: skipping organization_membership.created: missing user ID or organization ID", "id", event.ID)
			return
		}
		workosUser, err := wo.user(r.Context(), event.Data.UserID)
		if err != nil {
			if errors.Is(err, errUserNotFound) {
				slog.Info("WorkOS webhook: skipping organization_membership.created: WorkOS user not found", "id", event.ID, "workos_user", event.Data.UserID)
				return
			}
			slog.Error("WorkOS webhook error: failed to get WorkOS user", "id", event.ID, "workos_user", event.Data.UserID, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		orgID, err := wo.organizationExternalID(r.Context(), event.Data.OrganizationID)
		if err != nil {
			if errors.Is(err, errOrganizationNotLinked) {
				slog.Info("WorkOS webhook: skipping organization_membership.created: WorkOS organization doesn't have external ID", "id", event.ID, "workos_organization", event.Data.OrganizationID)
				return
			}
			if errors.Is(err, errOrganizationNotFound) {
				slog.Info("WorkOS webhook: skipping organization_membership.created: WorkOS organization not found", "id", event.ID, "workos_organization", event.Data.OrganizationID)
				return
			}
			slog.Error("WorkOS webhook error: failed to get WorkOS organization external ID", "id", event.ID, "workos_organization", event.Data.OrganizationID, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		org, err := wo.core.Organization(orgID)
		if err != nil {
			if _, ok := err.(*errors.NotFoundError); ok {
				slog.Info("WorkOS webhook: skipping organization_membership.created: organization not found", "id", event.ID, "organization", orgID)
				return
			}
			slog.Error("WorkOS webhook error: failed to get organization", "id", event.ID, "organization", orgID, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		email := strings.TrimSpace(norm.NFC.String(workosUser.Email))
		firstName := strings.TrimSpace(norm.NFC.String(workosUser.FirstName))
		lastName := strings.TrimSpace(norm.NFC.String(workosUser.LastName))
		name := firstName + " " + lastName
		if runes := []rune(name); len(runes) > 255 {
			name = string(runes[:255])
		}
		if _, err = org.AddMember(r.Context(), core.MemberToSet{Name: name, Email: email, WorkOSUserID: event.Data.UserID}); err != nil {
			if e, ok := err.(*errors.UnprocessableError); ok && e.Code == core.MemberEmailExists {
				slog.Info("WorkOS webhook: skipping organization_membership.created: member email already exists", "id", event.ID, "workos_user", event.Data.UserID, "organization", orgID)
				return
			}
			if e, ok := err.(*errors.UnprocessableError); ok && e.Code == core.MemberWorkOSUserIDExists {
				slog.Info("WorkOS webhook: skipping organization_membership.created: member WorkOS user ID already exists", "id", event.ID, "workos_user", event.Data.UserID, "organization", orgID)
				return
			}
			slog.Error("WorkOS webhook error: failed to provision member", "id", event.ID, "workos_user", event.Data.UserID, "organization", orgID, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		slog.Info("WorkOS webhook: member provisioned", "id", event.ID, "workos_user", event.Data.UserID, "organization", orgID)
	case "organization_membership.deleted":
		if event.Data.UserID == "" || event.Data.OrganizationID == "" {
			slog.Info("WorkOS webhook: skipping organization_membership.deleted: missing user ID or organization ID", "id", event.ID)
			return
		}
		orgID, err := wo.organizationExternalID(r.Context(), event.Data.OrganizationID)
		if err != nil {
			if errors.Is(err, errOrganizationNotLinked) {
				slog.Info("WorkOS webhook: skipping organization_membership.deleted: WorkOS organization doesn't have external ID", "id", event.ID, "workos_organization", event.Data.OrganizationID)
				return
			}
			if errors.Is(err, errOrganizationNotFound) {
				slog.Info("WorkOS webhook: skipping organization_membership.deleted: WorkOS organization not found", "id", event.ID, "workos_organization", event.Data.OrganizationID)
				return
			}
			slog.Error("WorkOS webhook error: failed to get WorkOS organization external ID", "id", event.ID, "workos_organization", event.Data.OrganizationID, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		org, err := wo.core.Organization(orgID)
		if err != nil {
			if _, ok := err.(*errors.NotFoundError); ok {
				slog.Info("WorkOS webhook: skipping organization_membership.deleted: organization not found", "id", event.ID, "organization", orgID)
				return
			}
			slog.Error("WorkOS webhook error: failed to get organization", "id", event.ID, "organization", orgID, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		memberID, err := org.MemberByWorkOSID(r.Context(), event.Data.UserID)
		if err != nil {
			if _, ok := err.(*errors.NotFoundError); ok {
				slog.Info("WorkOS webhook: skipping organization_membership.deleted: member not found", "id", event.ID, "workos_user", event.Data.UserID)
				return
			}
			slog.Error("WorkOS webhook error: failed to get member by WorkOS ID", "id", event.ID, "workos_user", event.Data.UserID, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if err := org.DeleteMember(r.Context(), memberID); err != nil {
			if _, ok := err.(*errors.NotFoundError); ok {
				slog.Info("WorkOS webhook: skipping organization_membership.deleted: member not found", "id", event.ID, "workos_user", event.Data.UserID)
				return
			}
			slog.Error("WorkOS webhook error: failed to delete member", "id", event.ID, "workos_user", event.Data.UserID, "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		slog.Info("WorkOS webhook: member deleted", "id", event.ID, "workos_user", event.Data.UserID, "organization", orgID)
	}
}
