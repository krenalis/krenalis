// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"fmt"
	"html"
	"maps"
	"net/smtp"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/metrics"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/util"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/jordan-wright/email"
	"golang.org/x/crypto/bcrypt"
)

var emailRegExp = regexp.MustCompile(`^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`)

// invitationTokenMaxAge represents the max age of an invitation token (3 days).
const invitationTokenMaxAge = 3 * 24 * 60 * 60

// resetPasswordTokenMaxAge represents the max age of a password token (1 hour).
const resetPasswordTokenMaxAge = 1 * 60 * 60

// Organization represents an organization.
type Organization struct {
	core         *Core
	organization *state.Organization
	ID           string `json:"id"`
	Name         string `json:"name"`

	// Enabled indicates whether the organization is enabled. Pipelines
	// belonging to a disabled organization behave as if they were disabled,
	// returning an OrganizationDisabled error wherever a PipelineDisabled error
	// would otherwise be returned. Event ingestion is the exception: requests
	// authenticated with the API key fail with an Unprocessable
	// OrganizationDisabled error, while those authenticated with an event write
	// key fail with a 503 Service Unavailable error.
	Enabled bool `json:"enabled"`
}

// Member represents a member of an organization.
type Member struct {
	ID         string    `json:"id"`
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

// MemberToSet represents a member to add or update.
type MemberToSet struct {
	Name         string  `json:"name"`     // Name, in range [0, 255]
	Email        string  `json:"email"`    // Email, in range [4,255] and must match `^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`
	Password     string  `json:"password"` // Password, at least 8 characters long; cannot be set when WorkOSUserID is set
	Avatar       *Avatar `json:"avatar"`
	WorkOSUserID string  `json:"workosUserID"` // WorkOS user ID; cannot be set when Password is set
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

// AccessKeyType represents an access key type.
type AccessKeyType int

const (
	AccessKeyTypeAPI AccessKeyType = iota
	AccessKeyTypeMCP
)

// MarshalJSON implements the json.Marshaler interface.
// It panics if mode is not a valid AccessKeyType value.
func (typ AccessKeyType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + typ.String() + `"`), nil
}

// String returns the string representation of typ.
// It panics if typ is not a valid AccessKeyType value.
func (typ AccessKeyType) String() string {
	switch typ {
	case AccessKeyTypeAPI:
		return "API"
	case AccessKeyTypeMCP:
		return "MCP"
	}
	panic("invalid access key type")
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (typ *AccessKeyType) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	m, ok := v.(string)
	if !ok {
		return fmt.Errorf("json: cannot scan a %T value into an AccessKeyType value", v)
	}
	var ty AccessKeyType
	switch m {
	case "API":
		ty = AccessKeyTypeAPI
	case "MCP":
		ty = AccessKeyTypeMCP
	default:
		return fmt.Errorf("json: invalid AccessKeyType: %s", m)
	}
	*typ = ty
	return nil
}

// IsValid reports whether typ is a valid AccessKeyType.
func (typ AccessKeyType) IsValid() bool {
	switch typ {
	case AccessKeyTypeAPI, AccessKeyTypeMCP:
		return true
	default:
		return false
	}
}

// AccessKey represents an access key.
type AccessKey struct {
	ID        string        `json:"id"`
	Workspace *string       `json:"workspace"`
	Name      string        `json:"name"`
	Type      AccessKeyType `json:"type"`
	Token     string        `json:"token"`
	CreatedAt time.Time     `json:"createdAt"`
}

// AddMember adds a new member of the organization and returns its identifier.
//
// Exactly one of member.Password and member.WorkOSUserID must be set.
//
// It returns an errors.UnprocessableError with code:
//
//   - MemberEmailExists, if a member with this email already exists in the
//     organization.
//   - MemberWorkOSUserIDExists, if a member with this WorkOS user ID already
//     exists in the organization.
//   - OrganizationNotExist, if the organization does not exist.
func (this *Organization) AddMember(ctx context.Context, member MemberToSet) (string, error) {
	this.core.mustBeOpen()
	if member.Password != "" && member.WorkOSUserID != "" {
		return "", errors.BadRequest("WorkOSUserID and password cannot be set at the same time")
	}
	err := validateMemberToSet(member, true, true, member.WorkOSUserID == "")
	if err != nil {
		return "", errors.BadRequest("%s", err)
	}

	password := member.Password
	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return "", err
		}
		password = string(hash)
	}

	n := state.AddMember{
		Organization: this.organization.ID,
	}
	for {
		n.ID = generateID(func(id string) (any, bool) {
			return nil, this.organization.HasMember(id)
		})
		now := time.Now().UTC()
		err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
			if member.Avatar != nil {
				_, err = tx.Exec(ctx,
					"INSERT INTO members (id, organization, name, avatar.image, avatar.mime_type, email, password, workos_user_id, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
					n.ID, this.organization.ID, member.Name, member.Avatar.Image, member.Avatar.MimeType, member.Email, password, member.WorkOSUserID, now)
			} else {
				_, err = tx.Exec(ctx,
					"INSERT INTO members (id, organization, name, avatar, email, password, workos_user_id, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
					n.ID, this.organization.ID, member.Name, nil, member.Email, password, member.WorkOSUserID, now)
			}
			if err != nil {
				if db.IsForeignKeyViolation(err) && db.ErrConstraintName(err) == "members_organization_fkey" {
					return nil, errors.Unprocessable(OrganizationNotExist, "organization %s does not exist", n.Organization)
				}
				if db.IsUniqueViolation(err) && db.ErrConstraintName(err) == "members_organization_email_key" {
					return nil, errors.Unprocessable(MemberEmailExists, "a member with this email already exists")
				}
				if db.IsUniqueViolation(err) && db.ErrConstraintName(err) == "members_workos_user_id_idx" {
					return nil, errors.Unprocessable(MemberWorkOSUserIDExists, "a member with this WorkOS user ID already exists")
				}
				return nil, err
			}
			return n, nil
		})
		if err != nil {
			if db.IsUniqueViolation(err) && db.ErrConstraintName(err) == "members_pkey" {
				continue
			}
			return "", err
		}
		break
	}
	return n.ID, nil
}

// AccessKeys returns the access keys of the organization ordered by creation
// time.
func (this *Organization) AccessKeys(ctx context.Context) ([]*AccessKey, error) {
	this.core.mustBeOpen()
	keys := make([]*AccessKey, 0)
	query := "SELECT id, workspace, name, type, hint, created_at FROM access_keys WHERE organization = $1 ORDER BY created_at"
	err := this.core.db.QueryScan(ctx, query, this.organization.ID, func(rows *db.Rows) error {
		var err error
		for rows.Next() {
			var key AccessKey
			var typ state.AccessKeyType
			if err = rows.Scan(&key.ID, &key.Workspace, &key.Name, &typ, &key.Token, &key.CreatedAt); err != nil {
				return err
			}
			key.Type = AccessKeyType(typ)
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
// and password. email's length must be in range [4, 255] and must be a valid
// email address. password's length must be at least 8 character long.
//
// If a member with the provided email does not exist or the password does not
// correspond, it returns an errors.UnprocessableError error with code
// AuthenticationFailed.
func (this *Organization) AuthenticateMember(ctx context.Context, email, password string) (string, error) {

	this.core.mustBeOpen()

	// Validate email.
	if err := util.ValidateStringField("email", email, 255); err != nil {
		return "", errors.BadRequest("%s", err)
	}
	if !emailRegExp.MatchString(email) {
		return "", errors.BadRequest("email is not a valid email address")
	}
	// Validate password.
	if password == "" {
		return "", errors.BadRequest("password is empty")
	}
	if !utf8.ValidString(password) {
		return "", errors.BadRequest("password is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(password); n < 8 {
		return "", errors.BadRequest("password must be at least 8 characters long")
	}

	var id string
	var hashedPassword []byte
	err := this.core.db.QueryRow(ctx, "SELECT id, password FROM members WHERE organization = $1 AND email = $2", this.organization.ID, email).Scan(&id, &hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.Unprocessable(AuthenticationFailed, "authentication has failed")
		}
		return "", err
	}
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		return "", errors.Unprocessable(AuthenticationFailed, "authentication has failed")
	}

	return id, nil
}

// CreateAccessKey creates a new access key for the organization with the
// specified name, which must be between 1 and 100 runes in length, and the
// specified type. If the workspace is not empty, the key will be restricted to
// that specific workspace. If the created key is an MCP key the workspace is
// required and therefore cannot be empty.
//
// It returns an errors.UnprocessableError error with code
//
//   - OrganizationNotExist, if the organization does not exist.
//   - WorkspaceNotExist, if the workspace does not exist.
func (this *Organization) CreateAccessKey(ctx context.Context, name, workspace string, typ AccessKeyType) (string, string, error) {

	this.core.mustBeOpen()

	if err := util.ValidateStringField("name", name, 100); err != nil {
		return "", "", errors.BadRequest("%s", err)
	}
	if workspace != "" && !IsValidID(workspace) {
		return "", "", errors.BadRequest("workspace %q is not a valid workspace identifier", workspace)
	}
	if workspace != "" {
		if _, ok := this.organization.Workspace(workspace); !ok {
			return "", "", errors.Unprocessable(WorkspaceNotExist, "workspace %s does not exist", workspace)
		}
	}
	if !typ.IsValid() {
		return "", "", errors.BadRequest("invalid access key type: %v", typ)
	}
	if typ == AccessKeyTypeMCP && workspace == "" {
		return "", "", errors.BadRequest("workspace is required for MCP keys")
	}
	// Generate a random access key,
	token, hmac, err := this.core.state.GenerateAccessKey(ctx)
	if err != nil {
		return "", "", err
	}
	defer clear(hmac)
	hint := token[:4] + "..." + token[len(token)-6:]
	n := state.CreateAccessKey{
		Organization: this.organization.ID,
		Workspace:    workspace,
		Type:         state.AccessKeyType(typ),
		HMAC:         hmac,
	}
	createdAt := time.Now().UTC().Truncate(time.Second)

	// Create the access key.
	for {
		n.ID = generateID[any](nil)
		err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
			_, err = tx.Exec(ctx, "INSERT INTO access_keys (id, organization, workspace, name, type, hmac, hint, created_at) "+
				"VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8)", n.ID, n.Organization, n.Workspace, name, n.Type, hmac, hint, createdAt)
			if err != nil {
				if db.IsForeignKeyViolation(err) {
					switch db.ErrConstraintName(err) {
					case "access_keys_organization_fkey":
						err = errors.Unprocessable(OrganizationNotExist, "organization %s does not exist", n.Organization)
					case "access_keys_workspace_fkey":
						err = errors.Unprocessable(WorkspaceNotExist, "workspace %s does not exist", n.Workspace)
					}
				}
				return nil, err
			}
			return n, nil
		})
		if err != nil {
			if db.IsUniqueViolation(err) && db.ErrConstraintName(err) == "access_keys_pkey" {
				continue
			}
			return "", "", err
		}
		break
	}

	return n.ID, token, nil
}

// Warehouse represents a data warehouse.
type Warehouse struct {
	Platform string        `json:"platform"`
	Mode     WarehouseMode `json:"mode"`
	Settings json.Value    `json:"settings"`
}

// CreateWorkspace creates a workspace with the given name, profile schema and
// displayed properties, and connects to a data warehouse of the given platform
// and settings.
// Returns the identifier of the workspace that has been created.
// name must be between 1 and 100 runes long.
//
// warehouse.Mode specifies the initial mode of the workspace's data warehouse.
//
// It returns an errors.UnprocessableError error with code:
//
//   - InvalidWarehouseSettings, if the warehouse settings are not valid.
//   - OrganizationNotExist, if the organization does not exist.
//   - WarehousePlatformNotExist, if a warehouse platform does not exist.
//   - WarehouseNotInitializable, if the warehouse is not initializable.
func (this *Organization) CreateWorkspace(ctx context.Context, name string, profileSchema types.Type, warehouse Warehouse, uiPreferences UIPreferences) (string, error) {

	this.core.mustBeOpen()

	settings, err := this.validateWorkspaceCreation(ctx, name, profileSchema, warehouse, uiPreferences)
	if err != nil {
		return "", err
	}

	// Initialize the data warehouse.
	err = this.core.datastore.Initialize(ctx, warehouse.Platform, settings, profileSchema)
	if err != nil {
		if err, ok := err.(*datastore.UnavailableError); ok {
			return "", errors.Unavailable("%s", err)
		}
		return "", err
	}

	n := state.CreateWorkspace{
		Organization:                   this.organization.ID,
		Name:                           name,
		ProfileSchema:                  profileSchema,
		ResolveIdentitiesOnBatchImport: true,
		UIPreferences:                  state.UIPreferences(uiPreferences),
	}
	n.Warehouse.Platform = warehouse.Platform
	n.Warehouse.Mode = state.WarehouseMode(warehouse.Mode)
	n.Warehouse.Settings, n.Warehouse.SettingsKey, err = this.core.state.EncryptSettings(ctx, settings)
	if err != nil {
		return "", err
	}
	n.Warehouse.MCPSettingsKey, err = this.core.state.GenerateKmsDataKey(ctx)
	if err != nil {
		return "", err
	}

	// Encode the profile schema to JSON.
	encodedProfileSchema, err := json.Marshal(n.ProfileSchema)
	if err != nil {
		return "", err
	}

	// Create the workspace.
	for {
		n.ID = generateID(this.organization.Workspace)
		err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
			_, err = tx.Exec(ctx, "INSERT INTO workspaces (id, organization, name,"+
				" profile_schema, resolve_identities_on_batch_import, ui_profile_image, ui_profile_first_name,"+
				" ui_profile_last_name, ui_profile_extra, warehouse_name, warehouse_mode,"+
				" warehouse_settings, kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key)"+
				" VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)",
				n.ID, n.Organization, n.Name, encodedProfileSchema, n.ResolveIdentitiesOnBatchImport,
				n.UIPreferences.Profile.Image, n.UIPreferences.Profile.FirstName,
				n.UIPreferences.Profile.LastName, n.UIPreferences.Profile.Extra, n.Warehouse.Platform,
				n.Warehouse.Mode, n.Warehouse.Settings, n.Warehouse.SettingsKey, n.Warehouse.MCPSettingsKey)
			if err != nil {
				if db.IsForeignKeyViolation(err) {
					if db.ErrConstraintName(err) == "workspaces_organization_fkey" {
						return nil, errors.Unprocessable(OrganizationNotExist, "organization %s does not exist", n.Organization)
					}
				}
				return nil, err
			}
			return n, nil
		})
		if err != nil {
			if db.IsUniqueViolation(err) && db.ErrConstraintName(err) == "workspaces_pkey" {
				continue
			}
			return "", err
		}
		break
	}

	return n.ID, nil
}

// DeleteAccessKey deletes the access key of the organization with identifier
// id. If the access key does not exist for the organization, it returns an
// errors.NotFound error.
func (this *Organization) DeleteAccessKey(ctx context.Context, id string) error {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return errors.BadRequest("identifier %q is not a valid access key identifier", id)
	}
	n := state.DeleteAccessKey{
		ID: id,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "DELETE FROM access_keys WHERE id = $1 AND organization = $2", id, this.organization.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("access key %s does not exist", id)
		}
		return n, nil
	})
	return err
}

// Delete deletes the organization.
func (this *Organization) Delete(ctx context.Context) error {
	this.core.mustBeOpen()
	n := state.DeleteOrganization{ID: this.organization.ID}
	return this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		// Mark the organization's pipeline functions as discontinued.
		now := time.Now().UTC()
		_, err := tx.Exec(ctx, "INSERT INTO discontinued_functions (id, discontinued_at)\n"+
			"SELECT p.transformation_id, $1\n"+
			"FROM pipelines AS p\n"+
			"INNER JOIN connections AS c ON p.connection = c.id\n"+
			"INNER JOIN workspaces AS w ON c.workspace = w.id\n"+
			"WHERE p.transformation_id != '' AND w.organization = $2\n"+
			"ON CONFLICT (id) DO NOTHING", now, n.ID)
		if err != nil {
			return nil, err
		}
		result, err := tx.Exec(ctx, "DELETE FROM organizations WHERE id = $1", this.organization.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("organization %q does not exist", this.organization.ID)
		}
		return n, nil
	})
}

// DeleteMember deletes a member of the organization with identifier id.
// If the member does not exist, it returns an errors.NotFound error.
func (this *Organization) DeleteMember(ctx context.Context, id string) error {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return errors.BadRequest("identifier %q is not a valid member identifier", id)
	}
	n := state.DeleteMember{
		ID:           id,
		Organization: this.organization.ID,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "DELETE FROM members WHERE id = $1 AND organization = $2", id, this.organization.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("member %s does not exist", id)
		}
		return n, nil
	})
	return err
}

// HasMember reports whether the organization has a member with the given ID.
func (this *Organization) HasMember(id string) (bool, error) {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return false, errors.BadRequest("identifier %q is not a valid member identifier", id)
	}
	return this.organization.HasMember(id), nil
}

// InviteMember sends an invitation email to the given email address using the
// given template. It then creates a new invited member, or updates an existing
// pending invitation.
//
// It returns an errors.UnprocessableError error with code
//   - EmailSendFailed, if emails cannot be sent.
//   - MemberEmailExists, if the email address belongs to an existing member.
func (this *Organization) InviteMember(ctx context.Context, email string, emailTemplate string) error {
	this.core.mustBeOpen()
	err := validateMemberEmail(email)
	if err != nil {
		return errors.BadRequest("%s", err)
	}
	if this.core.smtp == nil || this.core.memberEmailFrom == "" {
		return errors.Unprocessable(EmailSendFailed, "emails cannot be sent")
	}
	var invitationToken string
	for {
		id := generateID(func(id string) (any, bool) {
			return nil, this.organization.HasMember(id)
		})
		invitationToken, err = generateMemberToken()
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
			result, err := tx.Exec(ctx, "INSERT INTO members (id, organization, name, email, password, avatar, invitation_token, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) "+
				"ON CONFLICT (organization, email) DO UPDATE SET invitation_token = $7, created_at = $8 WHERE members.invitation_token <> ''",
				id, this.organization.ID, "", email, "", nil, invitationToken, now)
			if err != nil {
				return nil, err
			}
			if result.RowsAffected() == 0 {
				return nil, errors.Unprocessable(MemberEmailExists, "member with this email already exists")
			}
			return nil, nil
		})
		if err != nil {
			if db.IsUniqueViolation(err) && db.ErrConstraintName(err) == "members_pkey" {
				continue
			}
			if db.IsUniqueViolation(err) && db.ErrConstraintName(err) == "invitation_token_index" {
				continue
			}
			return err
		}
		break
	}
	t := strings.ReplaceAll(emailTemplate, "${token}", html.EscapeString(invitationToken))
	emailToSend := &emailToSend{
		From:     this.core.memberEmailFrom,
		Subject:  "You have been invited to Krenalis",
		To:       email,
		BodyHTML: []byte(t),
	}
	err = sendMail(emailToSend, this.core.smtp)
	return err
}

// Member returns the organization's member with identifier id.
// If the member does not exist, it returns an errors.NotFound error.
func (this *Organization) Member(ctx context.Context, id string) (*Member, error) {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return nil, errors.BadRequest("identifier %q is not a valid member identifier", id)
	}
	var member Member
	var avatarImage []byte
	var avatarMimeType *string
	var invitationToken string
	err := this.core.db.QueryRow(ctx, "SELECT id, name, email, (avatar).image, (avatar).mime_type, invitation_token, created_at FROM members WHERE id = $1 AND organization = $2", id, this.organization.ID).Scan(
		&member.ID, &member.Name, &member.Email, &avatarImage, &avatarMimeType, &invitationToken, &member.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("member %s does not exist", id)
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

// MemberByWorkOSID returns the ID of the member with the given WorkOS user ID
// in the organization. It returns an errors.NotFoundError if no member has that
// WorkOS user ID.
func (this *Organization) MemberByWorkOSID(ctx context.Context, workosUserID string) (string, error) {
	this.core.mustBeOpen()
	if len(workosUserID) == 0 {
		return "", errors.BadRequest("WorkOS user ID is empty")
	}
	var id string
	err := this.core.db.QueryRow(ctx, "SELECT id FROM members WHERE organization = $1 AND workos_user_id = $2", this.organization.ID, workosUserID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.NotFound("member with WorkOS user ID %s does not exist", workosUserID)
		}
		return "", err
	}
	return id, nil
}

// Members returns the organization's members sorted by name.
func (this *Organization) Members(ctx context.Context) ([]*Member, error) {
	this.core.mustBeOpen()
	members := []*Member{}
	err := this.core.db.QueryScan(ctx, "SELECT id, name, email, (avatar).image, (avatar).mime_type, invitation_token, created_at FROM members WHERE organization = $1 ORDER BY name", this.organization.ID, func(rows *db.Rows) error {
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

var errMemberNotFoundOrInvitationPending = errors.New("member not found or invitation pending")

// SendMemberPasswordReset sends a reset password email to the given email
// address using the given template.
//
// It returns an errors.UnprocessableError error with code
//   - EmailSendFailed, if emails cannot be sent.
func (this *Organization) SendMemberPasswordReset(ctx context.Context, email string, emailTemplate string) error {
	this.core.mustBeOpen()
	err := validateMemberEmail(email)
	if err != nil {
		return errors.BadRequest("%s", err)
	}
	if this.core.smtp == nil || this.core.memberEmailFrom == "" {
		return errors.Unprocessable(EmailSendFailed, "emails cannot be sent")
	}
	resetToken, err := generateMemberToken()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		exists, err := tx.QueryExists(ctx, "SELECT FROM members WHERE organization = $1 AND email = $2 AND invitation_token = ''", this.organization.ID, email)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errMemberNotFoundOrInvitationPending
		}
		_, err = tx.Exec(ctx, `UPDATE members SET reset_password_token = $1, reset_password_token_created_at = $2 WHERE organization = $3 AND email = $4`,
			resetToken, now, this.organization.ID, email)
		return nil, err
	})
	if err != nil {
		if err == errMemberNotFoundOrInvitationPending {
			// Do not return an error, to avoid revealing whether the email
			// belongs to an existing member or to a member with a pending invitation.
			return nil
		}
		return err
	}
	t := strings.ReplaceAll(emailTemplate, "${token}", html.EscapeString(resetToken))
	emailToSend := &emailToSend{
		From:     this.core.memberEmailFrom,
		Subject:  "Your Krenalis password reset",
		To:       email,
		BodyHTML: []byte(t),
	}
	err = sendMail(emailToSend, this.core.smtp)
	return err
}

// SetStatus sets the status of the organization.
func (this *Organization) SetStatus(ctx context.Context, enabled bool) error {
	this.core.mustBeOpen()

	if enabled == this.organization.Enabled {
		return nil
	}

	// Waits for the metrics to be saved.
	this.core.metrics.WaitStore()

	n := state.SetOrganizationStatus{
		ID:      this.organization.ID,
		Enabled: enabled,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE organizations SET enabled = $1 WHERE id = $2 AND enabled <> $1", n.Enabled, n.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, nil
		}
		// Terminate the organization's live pipeline runs.
		if !n.Enabled {
			endTime := time.Now().UTC()
			errorMessage := "organization has been disabled"
			// Mark the live runs as terminated and update the related pipeline state.
			_, err = tx.Exec(ctx, endOrganizationLiveRunsQuery,
				n.ID, endTime, errorMessage, metrics.TimeSlotFromTime(endTime), metrics.ReceiveStep)
			if err != nil {
				return nil, fmt.Errorf("cannot terminate organization live runs: %s", err)
			}
		}
		return n, nil
	})

	return err
}

// endOrganizationLiveRunsQuery ends all live pipeline runs for an organization.
// For each run it terminates, it records the final metrics, marks the pipeline
// as healthy, and records the termination error.
const endOrganizationLiveRunsQuery = `
WITH live_runs AS (
	SELECT r.id, r.pipeline
	FROM pipelines_runs AS r
	INNER JOIN pipelines AS p ON r.pipeline = p.id
	INNER JOIN connections AS c ON p.connection = c.id
	INNER JOIN workspaces AS w ON c.workspace = w.id
	WHERE w.organization = $1 AND r.end_time IS NULL
),
s AS (
	SELECT r.id,
		COALESCE(SUM(m.passed_0), 0) AS passed_0,
		COALESCE(SUM(m.passed_1), 0) AS passed_1,
		COALESCE(SUM(m.passed_2), 0) AS passed_2,
		COALESCE(SUM(m.passed_3), 0) AS passed_3,
		COALESCE(SUM(m.passed_4), 0) AS passed_4,
		COALESCE(SUM(m.passed_5), 0) AS passed_5,
		COALESCE(SUM(m.failed_0), 0) AS failed_0,
		COALESCE(SUM(m.failed_1), 0) AS failed_1,
		COALESCE(SUM(m.failed_2), 0) AS failed_2,
		COALESCE(SUM(m.failed_3), 0) AS failed_3,
		COALESCE(SUM(m.failed_4), 0) AS failed_4,
		COALESCE(SUM(m.failed_5), 0) AS failed_5
	FROM live_runs AS r
	LEFT JOIN pipelines_metrics AS m ON m.pipeline = r.pipeline
	GROUP BY r.id
),
ended_runs AS (
	UPDATE pipelines_runs AS r
	SET function = '',
		end_time = $2,
		passed_0 = r.passed_0 + s.passed_0,
		passed_1 = r.passed_1 + s.passed_1,
		passed_2 = r.passed_2 + s.passed_2,
		passed_3 = r.passed_3 + s.passed_3,
		passed_4 = r.passed_4 + s.passed_4,
		passed_5 = r.passed_5 + s.passed_5,
		failed_0 = r.failed_0 + s.failed_0,
		failed_1 = r.failed_1 + s.failed_1,
		failed_2 = r.failed_2 + s.failed_2,
		failed_3 = r.failed_3 + s.failed_3,
		failed_4 = r.failed_4 + s.failed_4,
		failed_5 = r.failed_5 + s.failed_5,
		error = $3
	FROM s
	WHERE r.id = s.id AND r.end_time IS NULL
	RETURNING r.pipeline
),
updated_pipelines AS (
	UPDATE pipelines AS p
	SET health = 'Healthy'::health
	FROM ended_runs
	WHERE p.id = ended_runs.pipeline
	RETURNING p.id
)
INSERT INTO pipelines_errors (pipeline, timeslot, step, count, message)
SELECT ended_runs.pipeline, $4, $5, 0, $3::text
FROM ended_runs
INNER JOIN updated_pipelines AS p ON ended_runs.pipeline = p.id
`

// TestWorkspaceCreation tests a workspace creation. It tests that a warehouse
// with the provided platform and settings can be initialized.
//
// It returns an errors.UnprocessableError error with code:
//
//   - InvalidWarehouseSettings, if the warehouse settings are not valid.
//   - WarehousePlatformNotExist, if a warehouse platform does not exist.
//   - WarehouseNotInitializable, if the warehouse intended for connection is
//     not initializable.
func (this *Organization) TestWorkspaceCreation(ctx context.Context, name string, profileSchema types.Type, warehouse Warehouse, uiPreferences UIPreferences) error {
	this.core.mustBeOpen()
	_, err := this.validateWorkspaceCreation(ctx, name, profileSchema, warehouse, uiPreferences)
	return err
}

// UpdateAccessKey updates the name of the access key for the organization with
// the specified identifier. name must be between 1 and 100 runes in length.
//
// If the access key does not exist for the organization, it returns an
// errors.NotFound error.
func (this *Organization) UpdateAccessKey(ctx context.Context, id, name string) error {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return errors.BadRequest("identifier %q is not a valid access key identifier", id)
	}
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return errors.BadRequest("%s", err)
	}
	result, err := this.core.db.Exec(ctx, "UPDATE access_keys SET name = $1 WHERE id = $2 AND organization = $3", name, id, this.organization.ID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.NotFound("access key %s does not exist", id)
	}
	return nil
}

// UpdateMember updates a member of the organization with identifier id.
// If password is empty, it does not change the password.
//
// If the member does not exist, it returns an errors.NotFound error. If the
// member to set has an email that is already used by another member, it returns
// an errors.UnprocessableError error with code MemberEmailExists.
func (this *Organization) UpdateMember(ctx context.Context, id string, member MemberToSet) error {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return errors.BadRequest("identifier %q is not a valid member identifier", id)
	}
	err := validateMemberToSet(member, true, true, false)
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
	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		exists, err := tx.QueryExists(ctx, "SELECT FROM members WHERE id = $1 AND organization = $2", id, this.organization.ID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.NotFound("member %s does not exist", id)
		}
		exists, err = tx.QueryExists(ctx, "SELECT FROM members WHERE id <> $1 AND organization = $2 AND email = $3", id, this.organization.ID, member.Email)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.Unprocessable(MemberEmailExists, "a member with this email already exists")
		}
		_, err = tx.Exec(ctx, "UPDATE members SET name = $1, email = $2 WHERE id = $3 AND organization = $4",
			member.Name, member.Email, id, this.organization.ID)
		if err != nil {
			return nil, err
		}
		if member.Avatar != nil {
			_, err = tx.Exec(ctx, "UPDATE members SET avatar.image = $1, avatar.mime_type = $2 WHERE id = $3 AND organization = $4",
				member.Avatar.Image, member.Avatar.MimeType, id, this.organization.ID)
		} else {
			_, err = tx.Exec(ctx, "UPDATE members SET avatar = $1 WHERE id = $2 AND organization = $3",
				nil, id, this.organization.ID)
		}
		if err != nil {
			return nil, err
		}
		if password != nil {
			_, err = tx.Exec(ctx, "UPDATE members SET password = $1 WHERE id = $2 AND organization = $3",
				string(password), id, this.organization.ID)
		}
		return nil, err
	})
	return err
}

// Update updates the name of the organization.
func (this *Organization) Update(ctx context.Context, name string) error {
	this.core.mustBeOpen()
	if err := util.ValidateStringField("name", name, 255); err != nil {
		return errors.BadRequest("%s", err)
	}
	n := state.UpdateOrganization{ID: this.organization.ID, Name: name}
	return this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE organizations SET name = $1 WHERE id = $2", name, this.organization.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("organization %q does not exist", this.organization.ID)
		}
		return n, nil
	})
}

// Workspace returns the organization's workspace with identifier id.
//
// It returns an errors.NotFound error if the workspace does not exist.
func (this *Organization) Workspace(id string) (*Workspace, error) {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return nil, errors.BadRequest("identifier %q is not a valid workspace identifier", id)
	}
	ws, ok := this.organization.Workspace(id)
	if !ok {
		return nil, errors.NotFound("workspace %s does not exist", id)
	}
	store, ok := this.core.datastore.Store(id)
	if !ok {
		return nil, errors.NotFound("workspace %s does not exist", id)
	}
	workspace := Workspace{
		core:                           this.core,
		organization:                   this,
		store:                          store,
		workspace:                      ws,
		ID:                             ws.ID,
		Name:                           ws.Name,
		ProfileSchema:                  ws.ProfileSchema,
		PrimarySources:                 maps.Clone(ws.PrimarySources),
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
	infos := make([]*Workspace, 0, len(workspaces))
	for _, ws := range workspaces {
		store, ok := this.core.datastore.Store(ws.ID)
		if !ok {
			continue
		}
		workspace := Workspace{
			core:                           this.core,
			organization:                   this,
			store:                          store,
			workspace:                      ws,
			ID:                             ws.ID,
			Name:                           ws.Name,
			ProfileSchema:                  ws.ProfileSchema,
			PrimarySources:                 maps.Clone(ws.PrimarySources),
			ResolveIdentitiesOnBatchImport: ws.ResolveIdentitiesOnBatchImport,
			Identifiers:                    ws.Identifiers,
			WarehouseMode:                  WarehouseMode(ws.Warehouse.Mode),
			UIPreferences:                  UIPreferences(ws.UIPreferences),
		}
		infos = append(infos, &workspace)
	}
	return infos
}

// validateWorkspaceCreation validates the arguments for a workspace creation.
// It tests that a warehouse with the provided platform and settings can be
// initialized, and returns an error if the arguments are not valid.
//
// It returns an errors.UnprocessableError error with code:
//
//   - InvalidWarehouseSettings, if the warehouse settings are not valid.
//   - WarehouseNotInitializable, if the warehouse intended for connection is
//     not initializable.
//   - WarehousePlatformNotExist, if a warehouse platform does not exist.
func (this *Organization) validateWorkspaceCreation(ctx context.Context, name string, profileSchema types.Type, warehouse Warehouse, uiPreferences UIPreferences) (json.Value, error) {

	// Validate the parameters.
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if !profileSchema.Valid() {
		return nil, errors.BadRequest("profile schema is invalid")
	}
	if err := validateUIPreferences(uiPreferences); err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if warehouse.Platform == "" {
		return nil, errors.BadRequest("warehouse platform is empty")
	}
	switch warehouse.Mode {
	case Normal, Inspection, Maintenance:
	default:
		return nil, errors.BadRequest("warehouse mode is not valid")
	}

	// Perform additional checks on the compliance of the profile schema.
	if err := checkAllowedPropertyProfileSchema(profileSchema); err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if err := datastore.CheckConflictingProperties("profile", profileSchema); err != nil {
		return nil, errors.BadRequest("%s", err)
	}

	// Validate the warehouse settings.
	settings, err := this.core.datastore.ValidateWarehouseSettings(ctx, warehouse.Platform, warehouse.Settings)
	if err != nil {
		if err == datastore.ErrWarehousePlatformNotExist {
			return nil, errors.Unprocessable(WarehousePlatformNotExist, "warehouse platform %s does not exist", warehouse.Platform)
		}
		if err, ok := err.(*warehouses.SettingsError); ok {
			return nil, errors.Unprocessable(InvalidWarehouseSettings, "data warehouse settings are not valid: %w", err.Err)
		}
		if err, ok := err.(*datastore.UnavailableError); ok {
			return nil, errors.Unavailable("%s", err)
		}
		return nil, err
	}

	// Check if the warehouse is initializable.
	err = this.core.datastore.CanInitialize(ctx, warehouse.Platform, settings)
	if err != nil {
		if err, ok := err.(*warehouses.NonInitializableError); ok {
			return nil, errors.Unprocessable(WarehouseNotInitializable, "cannot initialize the data warehouse: %w", err.Err)
		}
		if err, ok := err.(*datastore.UnavailableError); ok {
			return nil, errors.Unavailable("%s", err)
		}
		return nil, err
	}

	return settings, nil
}

const base62alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// generateEventWriteKeyToken generates a new event write key token.
func generateEventWriteKeyToken() string {
	// ⌈log₆₂ 2¹⁹²⌉ ≈ 32 chars
	src := make([]byte, 32)
	_, _ = rand.Read(src)
	for i := range src {
		src[i] = base62alphabet[src[i]%62]
	}
	return string(src)
}

// generateMemberToken generates a token that can be used for the
// invitation and for the password reset of a member.
func generateMemberToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// isInvitationTokenExpired checks if the invitation token of a member is
// expired, given the member's creation time.
func isInvitationTokenExpired(createdAt time.Time) bool {
	tokenExpiration := createdAt.Add(time.Duration(invitationTokenMaxAge) * time.Second)
	now := time.Now()
	return now.After(tokenExpiration)
}

// isResetPasswordTokenExpired checks if the reset password token of a
// member is expired, given the token's creation time.
func isResetPasswordTokenExpired(createdAt time.Time) bool {
	tokenExpiration := createdAt.Add(time.Duration(resetPasswordTokenMaxAge) * time.Second)
	now := time.Now().UTC()
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
	if config.Username != "" {
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
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
	if err := util.ValidateStringField("email", email, 255); err != nil {
		return err
	}
	if !emailRegExp.MatchString(email) {
		return errors.New("email is not a valid email address")
	}
	return nil
}

// validateMemberToSet validates a member to add or set and returns an error
// if the member is not valid.
func validateMemberToSet(member MemberToSet, validateName bool, validateEmail bool, validatePassword bool) error {
	// Validate name.
	if validateName && member.Name != "" {
		if err := util.ValidateStringField("name", member.Name, 255); err != nil {
			return err
		}
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
