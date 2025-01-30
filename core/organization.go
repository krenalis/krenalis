//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package core

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"fmt"
	"html"
	"maps"
	"math/big"
	"net/smtp"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/postgres"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/util"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/jordan-wright/email"
	"golang.org/x/crypto/bcrypt"
)

var emailRegExp = regexp.MustCompile(`^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`)

// invitationTokenMaxAge represents the max age of an invitation token (3 days).
const invitationTokenMaxAge = 3 * 24 * 60 * 60

// Organization represents an organization.
type Organization struct {
	core         *Core
	organization *state.Organization
	ID           int    `json:"id"`
	Name         string `json:"name"`
}

// Member represents a member of an organization.
type Member struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Avatar     *Avatar   `json:"avatar"`
	Invitation string    `json:"invitation"` // If the member has been invited, it is "Invited or "Expired"
	CreatedAt  time.Time `json:"createdAt"`
}

// Avatar represents an avatar of a member.
type Avatar struct {
	Image    []byte `json:"image"`    // Image, in range [1, 200000]
	MimeType string `json:"mimeType"` // Mime type, must be `image/jpeg` or `image/png`
}

// MemberToSet represents a member to update with the UpdateMember method.
type MemberToSet struct {
	Name     string  `json:"name"`     // Name, in range [1, 60]
	Email    string  `json:"email"`    // Email, in range [4,120] and must match `^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`
	Password string  `json:"password"` // Password, at least 8 characters long
	Avatar   *Avatar `json:"avatar"`
}

// emailToSend represents an email.
type emailToSend struct {
	RealName string
	From     string
	Subject  string
	To       string
	Cc       []string
	Bcc      []string
	BodyText []byte
	BodyHTML []byte
}

// APIKey represents an API key.
type APIKey struct {
	ID        int       `json:"id"`
	Workspace *int      `json:"workspace"`
	Name      string    `json:"name"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"createdAt"`
}

// APIKeys returns the API keys of the organization ordered by creation time.
func (this *Organization) APIKeys(ctx context.Context) ([]*APIKey, error) {
	this.core.mustBeOpen()
	keys := make([]*APIKey, 0)
	query := "SELECT id, workspace, name, token, created_at FROM api_keys WHERE organization = $1 ORDER BY created_at"
	err := this.core.db.QueryScan(ctx, query, this.organization.ID, func(rows *postgres.Rows) error {
		var err error
		for rows.Next() {
			var key APIKey
			if err = rows.Scan(&key.ID, &key.Workspace, &key.Name, &key.Token, &key.CreatedAt); err != nil {
				return err
			}
			if len(key.Token) != 43 {
				return fmt.Errorf("API key %d has an invalid token in the database", key.ID)
			}
			key.Token = key.Token[:4] + "..." + key.Token[40:]
			keys = append(keys, &key)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// AuthenticateMember authenticates a member of the organization given its email
// and password. email's length must be in range [4, 120] and must be a valid
// email address. password's length must be at least 8 character long.
//
// If a member with the provided email does not exist or the password does not
// correspond, it returns an errors.UnprocessableError error with code
// AuthenticationFailed.
func (this *Organization) AuthenticateMember(ctx context.Context, email, password string) (int, error) {

	this.core.mustBeOpen()

	// Validate email.
	if err := util.ValidateStringField("email", email, 120); err != nil {
		return 0, errors.BadRequest("%s", err)
	}
	if !emailRegExp.MatchString(email) {
		return 0, errors.BadRequest("email is not a valid email address")
	}
	// Validate password.
	if password == "" {
		return 0, errors.BadRequest("password is empty")
	}
	if !utf8.ValidString(password) {
		return 0, errors.BadRequest("password is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(password); n < 8 {
		return 0, errors.BadRequest("password must be at least 8 characters long")
	}

	var id int
	var hashedPassword []byte
	err := this.core.db.QueryRow(ctx, "SELECT id, password FROM members WHERE organization = $1 AND email = $2", this.organization.ID, email).Scan(&id, &hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.Unprocessable(AuthenticationFailed, "authentication has failed")
		}
		return 0, err
	}
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		return 0, errors.Unprocessable(AuthenticationFailed, "authentication has failed")
	}

	return id, nil
}

// CreateAPIKey creates a new API key for the organization with the specified
// name, which must be between 1 and 100 runes in length. If the workspace is
// not 0, the key will be restricted to that specific workspace.
//
// It returns an errors.UnprocessableError error with code
//
//   - OrganizationNotExist, if the organization does not exist.
//   - WorkspaceNotExist, if the workspace does not exist.
func (this *Organization) CreateAPIKey(ctx context.Context, name string, workspace int) (int, string, error) {
	this.core.mustBeOpen()
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return 0, "", errors.BadRequest("%s", err)
	}
	if workspace < 0 || workspace > maxInt32 {
		return 0, "", errors.BadRequest("workspace is not a valid workspace identifier")
	}
	if workspace > 0 {
		if _, ok := this.organization.Workspace(workspace); !ok {
			return 0, "", errors.Unprocessable(WorkspaceNotExist, "workspace %d does not exist", workspace)
		}
	}
	// Generate a random identifier.
	id, err := generateRandomID()
	if err != nil {
		return 0, "", err
	}
	n := state.CreateAPIKey{
		ID:           id,
		Organization: this.organization.ID,
		Workspace:    workspace,
		Token:        generateAPIKeyToken(),
	}
	createdAt := time.Now().UTC().Truncate(time.Second)
	err = this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO api_keys (id, organization, workspace, name, token, created_at) "+
			"VALUES ($1, $2, NULLIF($3, 0), $4, $5, $6)", n.ID, n.Organization, n.Workspace, name, n.Token, createdAt)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				switch postgres.ErrConstraintName(err) {
				case "api_keys_organization_fkey":
					err = errors.Unprocessable(OrganizationNotExist, "organization %d does not exist", n.Organization)
				case "api_keys_workspace_fkey":
					err = errors.Unprocessable(WorkspaceNotExist, "workspace %d does not exist", n.Workspace)
				}
			}
			return err
		}
		return tx.Notify(ctx, n)
	})
	if err != nil {
		return 0, "", err
	}
	return n.ID, n.Token, nil
}

// CreateWorkspace creates a workspace with the given name, user schema and
// displayed properties, and connects to a data warehouse of the provided name
// and settings. Returns the identifier of the workspace that has been created.
// name must be between 1 and 100 runes long.
//
// whMode specifies the initial mode of the workspace's data warehouse.
//
// It returns an errors.NotFoundError error if the organization does not exist
// anymore.
//
// It returns an errors.UnprocessableError error with code:
//
//   - InvalidWarehouseSettings, if the warehouse settings are not valid.
//   - WarehouseNonInitializable, if the warehouse is not initializable.
//   - WarehouseTypeNotExist, if a warehouse type does not exist.
func (this *Organization) CreateWorkspace(ctx context.Context, name string,
	userSchema types.Type, uiPreferences UIPreferences,
	whType string, whSettings []byte, whMode WarehouseMode) (int, error) {

	this.core.mustBeOpen()

	whSettings, err := this.validateWorkspaceCreation(ctx, name, userSchema, uiPreferences, whType, whSettings, whMode)
	if err != nil {
		return 0, err
	}

	// Initialize the data warehouse.
	err = this.core.datastore.Initialize(ctx, whType, whSettings, userSchema)
	if err != nil {
		if err, ok := err.(*datastore.UnavailableError); ok {
			return 0, errors.Unavailable("%s", err)
		}
		return 0, err
	}

	n := state.CreateWorkspace{
		Organization:                   this.organization.ID,
		Name:                           name,
		UserSchema:                     userSchema,
		ResolveIdentitiesOnBatchImport: true,
		UIPreferences:                  state.UIPreferences(uiPreferences),
	}
	n.Warehouse.Type = whType
	n.Warehouse.Mode = state.WarehouseMode(whMode)
	n.Warehouse.Settings = whSettings

	// Generate the identifier.
	n.ID, err = generateRandomID()
	if err != nil {
		return 0, err
	}

	// Encode the user schema to JSON.
	encodedUserSchema, err := json.Marshal(n.UserSchema)
	if err != nil {
		return 0, err
	}

	err = this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO workspaces (id, organization, name,"+
			" user_schema, resolve_identities_on_batch_import, ui_user_profile_image, ui_user_profile_first_name, "+
			" ui_user_profile_last_name, ui_user_profile_extra, warehouse_type, warehouse_mode, warehouse_settings)"+
			" VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)",
			n.ID, n.Organization, n.Name, encodedUserSchema, n.ResolveIdentitiesOnBatchImport,
			n.UIPreferences.UserProfile.Image, n.UIPreferences.UserProfile.FirstName,
			n.UIPreferences.UserProfile.LastName, n.UIPreferences.UserProfile.Extra,
			n.Warehouse.Type, n.Warehouse.Mode, n.Warehouse.Settings)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "workspaces_organization_fkey" {
					return errors.NotFound("organization %d does not exist", n.Organization)
				}
			}
			return err
		}
		return tx.Notify(ctx, n)
	})
	if err != nil {
		return 0, err
	}

	return n.ID, nil
}

// DeleteAPIKey deletes the API key of the organization with identifier id.
// If the API key does not exist for the organization, it returns an
// errors.NotFound error.
func (this *Organization) DeleteAPIKey(ctx context.Context, id int) error {
	this.core.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return errors.BadRequest("identifier %d is not a valid API key identifier", id)
	}
	n := state.DeleteAPIKey{
		ID: id,
	}
	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "DELETE FROM api_keys WHERE id = $1 AND organization = $2", id, this.organization.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("API key %d does not exist", id)
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// DeleteMember deletes a member of the organization with identifier id.
// If the member does not exist, it returns an errors.NotFound error.
func (this *Organization) DeleteMember(ctx context.Context, id int) error {
	this.core.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return errors.BadRequest("identifier %d is not a valid member identifier", id)
	}
	result, err := this.core.db.Exec(ctx, "DELETE FROM members WHERE id = $1 AND organization = $2", id, this.organization.ID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.NotFound("member %d does not exist", id)
	}
	return nil
}

// InviteMember sends an invitation email to the given email address using the
// given template. It then creates a new invited member.
//
// It returns an errors.UnprocessableError error with code
//   - EmailSendFailed, if emails cannot be sent.
//   - MemberEmailExists, if the email address has already been invited.
func (this *Organization) InviteMember(ctx context.Context, email string, emailTemplate string) error {
	this.core.mustBeOpen()
	err := validateMemberEmail(email)
	if err != nil {
		return errors.BadRequest("%s", err)
	}
	invitationToken, err := generateInvitationToken()
	if err != nil {
		return err
	}
	if this.core.smtp == nil {
		return errors.Unprocessable(EmailSendFailed, "emails cannot be sent")
	}
	now := time.Now().UTC()
	err = this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		err := this.core.db.QueryVoid(ctx, "SELECT FROM members WHERE organization = $1 AND email = $2 AND invitation_token = ''", this.organization.ID, email)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil {
			return errors.Unprocessable(MemberEmailExists, "a member with this email already exists")
		}
		_, err = this.core.db.Exec(ctx, "INSERT INTO members (organization, name, email, password, avatar, invitation_token, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7) "+
			"ON CONFLICT (organization, email) DO UPDATE SET invitation_token = $6, created_at = $7",
			this.organization.ID, "", email, "", nil, invitationToken, now)
		return err
	})
	if err != nil {
		return err
	}
	t := strings.ReplaceAll(emailTemplate, "${token}", html.EscapeString(invitationToken))
	emailToSend := &emailToSend{
		From:     this.core.smtp.User,
		Subject:  "You have been invited to Meergo",
		To:       email,
		BodyHTML: []byte(t),
	}
	err = sendMail(emailToSend, this.core.smtp)
	return err
}

// Member returns the organization's member with identifier id.
// If the member does not exist, it returns an errors.NotFound error.
func (this *Organization) Member(ctx context.Context, id int) (*Member, error) {
	this.core.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid member identifier", id)
	}
	var member Member
	var avatarImage []byte
	var avatarMimeType *string
	var invitationToken string
	err := this.core.db.QueryRow(ctx, "SELECT id, name, email, (avatar).image, (avatar).mime_type, invitation_token, created_at FROM members WHERE id = $1 AND organization = $2", id, this.organization.ID).Scan(
		&member.ID, &member.Name, &member.Email, &avatarImage, &avatarMimeType, &invitationToken, &member.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("member %d does not exist", id)
		}
		return nil, err
	}
	if len(avatarImage) > 0 {
		member.Avatar = &Avatar{
			Image:    avatarImage,
			MimeType: *avatarMimeType,
		}
	}
	if invitationToken != "" {
		member.Invitation = "Invited"
		if isInvitationTokenExpired(member.CreatedAt) {
			member.Invitation = "Expired"
		}
	}
	return &member, nil
}

// Members returns the organization's members sorted by name.
func (this *Organization) Members(ctx context.Context) ([]*Member, error) {
	this.core.mustBeOpen()
	members := []*Member{}
	err := this.core.db.QueryScan(ctx, "SELECT id, name, email, (avatar).image, (avatar).mime_type, invitation_token, created_at FROM members WHERE organization = $1 ORDER BY name", this.organization.ID, func(rows *postgres.Rows) error {
		var err error
		for rows.Next() {
			var member Member
			var avatarImage []byte
			var avatarMimeType *string
			var invitationToken string
			if err = rows.Scan(&member.ID, &member.Name, &member.Email, &avatarImage, &avatarMimeType, &invitationToken, &member.CreatedAt); err != nil {
				return err
			}
			if len(avatarImage) > 0 {
				member.Avatar = &Avatar{
					Image:    avatarImage,
					MimeType: *avatarMimeType,
				}
			}
			if invitationToken != "" {
				member.Invitation = "Invited"
				if isInvitationTokenExpired(member.CreatedAt) {
					member.Invitation = "Expired"
				}
			}
			members = append(members, &member)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return members, nil
}

// TestWorkspaceCreation tests a workspace creation. It tests that a warehouse
// with the provided name and settings can be initialized.
//
// It returns an errors.UnprocessableError error with code:
//
//   - InvalidWarehouseSettings, if the warehouse settings are not valid.
//   - WarehouseNonInitializable, if the warehouse intended for connection is
//     not initializable.
//   - WarehouseTypeNotExist, if a warehouse type does not exist.
func (this *Organization) TestWorkspaceCreation(ctx context.Context, name string,
	userSchema types.Type, uiPreferences UIPreferences, whType string,
	whSettings []byte, mode WarehouseMode) error {
	this.core.mustBeOpen()
	_, err := this.validateWorkspaceCreation(ctx, name, userSchema, uiPreferences, whType, whSettings, mode)
	return err
}

// UpdateAPIKey updates the name of the API key for the organization with the
// specified identifier. name must be between 1 and 100 runes in length.
//
// If the API key does not exist for the organization, it returns an
// errors.NotFound error.
func (this *Organization) UpdateAPIKey(ctx context.Context, id int, name string) error {
	this.core.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return errors.BadRequest("identifier %d is not a valid API key identifier", id)
	}
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return errors.BadRequest("%s", err)
	}
	result, err := this.core.db.Exec(ctx, "UPDATE api_keys SET name = $1 WHERE id = $2 AND organization = $3", name, id, this.organization.ID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.NotFound("API key %d does not exist", id)
	}
	return nil
}

// UpdateMember updates a member of the organization with identifier id.
// If password is empty, it does not change the password.
//
// If the member does not exist, it returns an errors.NotFound error. If the
// member to set has an email that is already used by another member, it returns
// an errors.UnprocessableError error with code MemberEmailExists.
func (this *Organization) UpdateMember(ctx context.Context, id int, member MemberToSet) error {
	this.core.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return errors.BadRequest("identifier %d is not a valid member identifier", id)
	}
	err := validateMemberToSet(member, true, false)
	if err != nil {
		return errors.BadRequest("%s", err)
	}
	var password []byte
	if member.Password != "" {
		password, err = bcrypt.GenerateFromPassword([]byte(member.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
	}
	err = this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		err := this.core.db.QueryVoid(ctx, "SELECT FROM members WHERE id = $1 AND organization = $2", id, this.organization.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.NotFound("member %d does not exist", id)
			}
			return err
		}
		err = this.core.db.QueryVoid(ctx, "SELECT FROM members WHERE id <> $1 AND organization = $2 AND email = $3", id, this.organization.ID, member.Email)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err != sql.ErrNoRows {
			return errors.Unprocessable(MemberEmailExists, "a member with this email already exists")
		}
		_, err = this.core.db.Exec(ctx, "UPDATE members SET name = $1, email = $2 WHERE id = $3 AND organization = $4",
			member.Name, member.Email, id, this.organization.ID)
		if err != nil {
			return err
		}
		if member.Avatar != nil {
			_, err = this.core.db.Exec(ctx, "UPDATE members SET avatar.image = $1, avatar.mime_type = $2 WHERE id = $3 AND organization = $4",
				member.Avatar.Image, member.Avatar.MimeType, id, this.organization.ID)
		} else {
			_, err = this.core.db.Exec(ctx, "UPDATE members SET avatar = $1 WHERE id = $2 AND organization = $3",
				nil, id, this.organization.ID)
		}
		if err != nil {
			return err
		}
		if password != nil {
			_, err = this.core.db.Exec(ctx, "UPDATE members SET password = $1 WHERE id = $2 AND organization = $3",
				string(password), id, this.organization.ID)
		}
		return err
	})
	return err
}

// Workspace returns the organization's workspace with identifier id.
//
// It returns an errors.NotFound error if the workspace does not exist.
func (this *Organization) Workspace(id int) (*Workspace, error) {
	this.core.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid workspace identifier", id)
	}
	ws, ok := this.organization.Workspace(id)
	if !ok {
		return nil, errors.NotFound("workspace %d does not exist", id)
	}
	workspace := Workspace{
		core:                           this.core,
		organization:                   this,
		store:                          this.core.datastore.Store(id),
		workspace:                      ws,
		ID:                             ws.ID,
		Name:                           ws.Name,
		UserSchema:                     ws.UserSchema,
		UserPrimarySources:             maps.Clone(ws.UserPrimarySources),
		ResolveIdentitiesOnBatchImport: ws.ResolveIdentitiesOnBatchImport,
		Identifiers:                    ws.Identifiers,
		WarehouseMode:                  WarehouseMode(ws.Warehouse.Mode),
		UIPreferences:                  UIPreferences(ws.UIPreferences),
	}
	return &workspace, nil
}

// Workspaces returns the workspaces of the organization.
func (this *Organization) Workspaces() []*Workspace {
	this.core.mustBeOpen()
	workspaces := this.organization.Workspaces()
	infos := make([]*Workspace, len(workspaces))
	for i, ws := range workspaces {
		workspace := Workspace{
			core:                           this.core,
			organization:                   this,
			store:                          this.core.datastore.Store(ws.ID),
			workspace:                      ws,
			ID:                             ws.ID,
			Name:                           ws.Name,
			UserSchema:                     ws.UserSchema,
			UserPrimarySources:             maps.Clone(ws.UserPrimarySources),
			ResolveIdentitiesOnBatchImport: ws.ResolveIdentitiesOnBatchImport,
			Identifiers:                    ws.Identifiers,
			WarehouseMode:                  WarehouseMode(ws.Warehouse.Mode),
			UIPreferences:                  UIPreferences(ws.UIPreferences),
		}
		infos[i] = &workspace
	}
	return infos
}

// validateWorkspaceCreation validates the arguments for a workspace creation.
// It tests that a warehouse with the provided name and settings can be
// initialized, and returns an error if the arguments are not valid.
//
// It returns an errors.UnprocessableError error with code:
//
//   - InvalidWarehouseSettings, if the warehouse settings are not valid.
//   - WarehouseNonInitializable, if the warehouse intended for connection is
//     not initializable.
//   - WarehouseTypeNotExist, if a warehouse type does not exist.
func (this *Organization) validateWorkspaceCreation(ctx context.Context, name string,
	userSchema types.Type, uiPreferences UIPreferences,
	whType string, whSettings []byte, whMode WarehouseMode) ([]byte, error) {

	// Validate the parameters.
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if !userSchema.Valid() {
		return nil, errors.BadRequest("user schema is invalid")
	}
	if err := validateUIPreferences(uiPreferences); err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if whType == "" {
		return nil, errors.BadRequest("warehouse type is empty")
	}
	switch whMode {
	case Normal, Inspection, Maintenance:
	default:
		return nil, errors.BadRequest("warehouse mode is not valid")
	}

	// Perform additional checks on the compliance of the user schema.
	if err := checkAllowedPropertyUserSchema(userSchema); err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if err := datastore.CheckConflictingProperties("users", userSchema); err != nil {
		return nil, errors.BadRequest("%s", err)
	}

	// Normalize the warehouse settings.
	settings, err := this.core.datastore.NormalizeWarehouseSettings(whType, whSettings)
	if err != nil {
		if err == datastore.WarehouseTypeNotExist {
			return nil, errors.Unprocessable(WarehouseTypeNotExist, "warehouse type %q does not exist", whType)
		}
		if err, ok := err.(*meergo.WarehouseSettingsError); ok {
			return nil, errors.Unprocessable(InvalidWarehouseSettings, "data warehouse settings are not valid: %w", err.Err)
		}
		return nil, err
	}

	// Check if the warehouse is initializable.
	err = this.core.datastore.CanInitialize(ctx, whType, settings)
	if err != nil {
		if err, ok := err.(*meergo.WarehouseNonInitializableError); ok {
			return nil, errors.Unprocessable(WarehouseNonInitializable, "data warehouse is not initializable: %w", err.Err)
		}
		if err, ok := err.(*datastore.UnavailableError); ok {
			return nil, errors.Unavailable("%s", err)
		}
		return nil, err
	}

	return settings, nil
}

// generateAPIKeyToken generates a new API key token.
func generateAPIKeyToken() string {
	// ⌈log₆₂ 2²⁵⁶⌉ ≈ 43 chars
	const base62alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	src := make([]byte, 43)
	_, _ = rand.Read(src)
	for i := range src {
		src[i] = base62alphabet[src[i]%62]
	}
	return string(src)
}

// generateInvitationToken generates an invitation token.
func generateInvitationToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

var bigMaxInt32 = big.NewInt(maxInt32)

// generateRandomID generates a random identifier in [1, maxInt32].
func generateRandomID() (int, error) {
	n, err := rand.Int(rand.Reader, bigMaxInt32)
	if err != nil {
		return 0, err
	}
	return int(n.Int64()) + 1, nil
}

// isInvitationTokenExpired checks if the invitation token of a member is expired, given
// the member's creation time.
func isInvitationTokenExpired(createdAt time.Time) bool {
	tokenExpiration := createdAt.Add(time.Duration(invitationTokenMaxAge) * time.Second)
	now := time.Now()
	return now.After(tokenExpiration)
}

func sendMail(mail *emailToSend, config *SMTPConfig) error {
	e := email.NewEmail()
	if mail.RealName != "" {
		e.From = mail.RealName + "<" + mail.From + ">"
	} else {
		e.From = mail.From
	}
	e.To = []string{mail.To}
	e.Subject = mail.Subject
	if len(mail.Cc) > 0 {
		e.Cc = mail.Cc
	}
	if len(mail.Bcc) > 0 {
		e.Bcc = mail.Bcc
	}
	if mail.BodyText != nil {
		e.Text = mail.BodyText
	}
	if mail.BodyHTML != nil {
		e.HTML = mail.BodyHTML
	}
	e.Headers = textproto.MIMEHeader{"X-Mailer": []string{"Open2b"}}
	var auth smtp.Auth
	if config.User != "" {
		auth = smtp.PlainAuth("", config.User, config.Pass, config.Host)
	}
	conf := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         config.Host,
	}
	var err error
	if config.Port == 465 {
		err = e.SendWithTLS(config.Host+":"+strconv.Itoa(config.Port), auth, conf)
	} else {
		err = e.SendWithStartTLS(config.Host+":"+strconv.Itoa(config.Port), auth, conf)
		if err2, ok := err.(x509.HostnameError); ok {
			if len(err2.Certificate.DNSNames) > 0 {
				hostname := err2.Certificate.DNSNames[0]
				err = e.SendWithStartTLS(hostname+":"+strconv.Itoa(config.Port), auth, conf)
			}
		}
	}
	return err
}

// validateMemberEmail validates a member's email and returns an error if it is
// not valid.
func validateMemberEmail(email string) error {
	if err := util.ValidateStringField("email", email, 120); err != nil {
		return err
	}
	if !emailRegExp.MatchString(email) {
		return errors.New("email is not a valid email address")
	}
	return nil
}

// validateMemberToSet validates a member to add or set and returns an error
// if the member is not valid.
func validateMemberToSet(member MemberToSet, validateEmail bool, validatePassword bool) error {
	// Validate name.
	if err := util.ValidateStringField("name", member.Name, 45); err != nil {
		return err
	}
	// Validate avatar.
	if member.Avatar != nil {
		if member.Avatar.MimeType != "image/jpeg" && member.Avatar.MimeType != "image/png" {
			return errors.New("image must be in jpeg or png format")
		}
		if len(member.Avatar.Image) > 200*1024 {
			return errors.New("image is bigger than 200kb")
		}
	}
	// Validate email.
	if validateEmail {
		err := validateMemberEmail(member.Email)
		if err != nil {
			return err
		}
	}
	// Validate password.
	if validatePassword && member.Password == "" {
		return errors.New("password is empty")
	}
	if member.Password != "" {
		if !utf8.ValidString(member.Password) {
			return errors.New("password is not UTF-8 encoded")
		}
		if n := utf8.RuneCountInString(member.Password); n < 8 {
			return errors.New("password must be at least 8 characters long")
		}
		if n := utf8.RuneCountInString(member.Password); n > 72 {
			return errors.New("password is longer than 72 runes")
		}
	}
	return nil
}
