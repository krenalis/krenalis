// Copyright 2025 Open2b. All rights reserved.
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
	"math/big"
	"net/smtp"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/internal/datastore"
	"github.com/meergo/meergo/core/internal/db"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/util"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/warehouses"

	"github.com/jordan-wright/email"
	"golang.org/x/crypto/bcrypt"
)

var emailRegExp = regexp.MustCompile(`^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`)

// invitationTokenMaxAge represents the max age of an invitation token (3 days).
const invitationTokenMaxAge = 3 * 24 * 60 * 60

// resetPasswordTokenMaxAge represents the max age of a password token (1 hour).
const resetPasswordTokenMaxAge = 1 * 60 * 60

var errResetPasswordTokenNotExist = errors.New("The reset password token doesn't exist")

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
	ID        int           `json:"id"`
	Workspace *int          `json:"workspace"`
	Name      string        `json:"name"`
	Type      AccessKeyType `json:"type"`
	Token     string        `json:"token"`
	CreatedAt time.Time     `json:"createdAt"`
}

// AddMember adds a new member of the organization.
//
// If the member to add has an email that is already used by another
// member, it returns an errors.UnprocessableError error with code
// MemberEmailExists.
func (this *Organization) AddMember(ctx context.Context, member MemberToSet) error {
	this.core.mustBeOpen()
	err := validateMemberToSet(member, true, true, true)
	if err != nil {
		return errors.BadRequest("%s", err)
	}
	password, err := bcrypt.GenerateFromPassword([]byte(member.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	n := state.AddMember{
		Organization: this.organization.ID,
	}
	now := time.Now().UTC()
	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		exists, err := tx.QueryExists(ctx, "SELECT FROM members WHERE organization = $1 AND email = $2", this.organization.ID, member.Email)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.Unprocessable(MemberEmailExists, "a member with this email already exists")
		}
		if member.Avatar != nil {
			err = tx.QueryRow(ctx,
				"INSERT INTO members (name, email, password, avatar.image, avatar.mime_type, organization, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id",
				member.Name, member.Email, password, member.Avatar.Image, member.Avatar.MimeType, this.organization.ID, now).Scan(&n.ID)
		} else {
			err = tx.QueryRow(ctx,
				"INSERT INTO members (name, email, password, avatar, organization, created_at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
				member.Name, member.Email, password, nil, this.organization.ID, now).Scan(&n.ID)
		}
		if err != nil {
			return nil, err
		}
		return n, nil
	})
	return err
}

// AccessKeys returns the access keys of the organization ordered by creation
// time.
func (this *Organization) AccessKeys(ctx context.Context) ([]*AccessKey, error) {
	this.core.mustBeOpen()
	keys := make([]*AccessKey, 0)
	query := "SELECT id, workspace, name, type, token, created_at FROM access_keys WHERE organization = $1 ORDER BY created_at"
	err := this.core.db.QueryScan(ctx, query, this.organization.ID, func(rows *db.Rows) error {
		var err error
		for rows.Next() {
			var key AccessKey
			var typ state.AccessKeyType
			if err = rows.Scan(&key.ID, &key.Workspace, &key.Name, &typ, &key.Token, &key.CreatedAt); err != nil {
				return err
			}
			key.Type = AccessKeyType(typ)
			if len(key.Token) != 43 {
				return fmt.Errorf("access key %d has an invalid token in the database", key.ID)
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

// CreateAccessKey creates a new access key for the organization with the
// specified name, which must be between 1 and 100 runes in length, and the
// specified type. If the workspace is not 0, the key will be restricted to that
// specific workspace. If the created key is an MCP key the workspace is
// required and therefore cannot be 0.
//
// It returns an errors.UnprocessableError error with code
//
//   - OrganizationNotExist, if the organization does not exist.
//   - WorkspaceNotExist, if the workspace does not exist.
func (this *Organization) CreateAccessKey(ctx context.Context, name string, workspace int, typ AccessKeyType) (int, string, error) {
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
	if !typ.IsValid() {
		return 0, "", errors.BadRequest("invalid access key type: %v", typ)
	}
	if typ == AccessKeyTypeMCP && workspace == 0 {
		return 0, "", errors.BadRequest("workspace is required for MCP keys")
	}
	// Generate a random identifier.
	id, err := generateRandomID()
	if err != nil {
		return 0, "", err
	}
	n := state.CreateAccessKey{
		ID:           id,
		Organization: this.organization.ID,
		Workspace:    workspace,
		Type:         state.AccessKeyType(typ),
		Token:        generateAccessKeyToken(),
	}
	createdAt := time.Now().UTC().Truncate(time.Second)
	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		_, err := tx.Exec(ctx, "INSERT INTO access_keys (id, organization, workspace, name, type, token, created_at) "+
			"VALUES ($1, $2, NULLIF($3, 0), $4, $5, $6, $7)", n.ID, n.Organization, n.Workspace, name, typ, n.Token, createdAt)
		if err != nil {
			if db.IsForeignKeyViolation(err) {
				switch db.ErrConstraintName(err) {
				case "access_keys_organization_fkey":
					err = errors.Unprocessable(OrganizationNotExist, "organization %d does not exist", n.Organization)
				case "access_keys_workspace_fkey":
					err = errors.Unprocessable(WorkspaceNotExist, "workspace %d does not exist", n.Workspace)
				}
			}
			return nil, err
		}
		return n, nil
	})
	if err != nil {
		return 0, "", err
	}
	return n.ID, n.Token, nil
}

// CreateWorkspace creates a workspace with the given name, user schema and
// displayed properties, and connects to a data warehouse of the provided driver
// name, settings and MCP settings (which can be nil, meaning that they are not
// configured for the workspace).
// Returns the identifier of the workspace that has been created.
// name must be between 1 and 100 runes long.
//
// whMode specifies the initial mode of the workspace's data warehouse.
//
// It returns an errors.UnprocessableError error with code:
//
//   - InvalidWarehouseSettings, if the warehouse settings are not valid.
//   - NotReadOnlyMCPSettings, if the MCP settings do not grant access to a
//     read-only user on the data warehouse.
//   - OrganizationNotExist, if the organization does not exist.
//   - WarehouseDriverNotExist, if a warehouse driver does not exist.
//   - WarehouseNonInitializable, if the warehouse is not initializable.
func (this *Organization) CreateWorkspace(ctx context.Context, name string,
	userSchema types.Type, uiPreferences UIPreferences,
	whName string, whSettings, whMCPSettings []byte, whMode WarehouseMode) (int, error) {

	this.core.mustBeOpen()

	whSettings, whMCPSettings, err := this.validateWorkspaceCreation(ctx, name,
		userSchema, uiPreferences, whName, whSettings, whMCPSettings, whMode)
	if err != nil {
		return 0, err
	}

	// Initialize the data warehouse.
	err = this.core.datastore.Initialize(ctx, whName, whSettings, userSchema)
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
	n.Warehouse.Name = whName
	n.Warehouse.Mode = state.WarehouseMode(whMode)
	n.Warehouse.Settings = whSettings
	n.Warehouse.MCPSettings = whMCPSettings

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

	var mcp string
	if n.Warehouse.MCPSettings != nil {
		mcp = string(n.Warehouse.MCPSettings)
	} else {
		mcp = "null"
	}
	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		_, err := tx.Exec(ctx, "INSERT INTO workspaces (id, organization, name,"+
			" user_schema, resolve_identities_on_batch_import, ui_user_profile_image, ui_user_profile_first_name, "+
			" ui_user_profile_last_name, ui_user_profile_extra, warehouse_name, "+
			"warehouse_mode, warehouse_settings, warehouse_mcp_settings)"+
			" VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)",
			n.ID, n.Organization, n.Name, encodedUserSchema, n.ResolveIdentitiesOnBatchImport,
			n.UIPreferences.UserProfile.Image, n.UIPreferences.UserProfile.FirstName,
			n.UIPreferences.UserProfile.LastName, n.UIPreferences.UserProfile.Extra,
			n.Warehouse.Name, n.Warehouse.Mode, n.Warehouse.Settings, mcp)
		if err != nil {
			if db.IsForeignKeyViolation(err) {
				if db.ErrConstraintName(err) == "workspaces_organization_fkey" {
					return nil, errors.Unprocessable(OrganizationNotExist, "organization %d does not exist", n.Organization)
				}
			}
			return nil, err
		}
		return n, nil
	})
	if err != nil {
		return 0, err
	}

	return n.ID, nil
}

// DeleteAccessKey deletes the access key of the organization with identifier
// id. If the access key does not exist for the organization, it returns an
// errors.NotFound error.
func (this *Organization) DeleteAccessKey(ctx context.Context, id int) error {
	this.core.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return errors.BadRequest("identifier %d is not a valid access key identifier", id)
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
			return nil, errors.NotFound("access key %d does not exist", id)
		}
		return n, nil
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
	if !this.organization.HasMember(id) {
		return errors.NotFound("member %d does not exist", id)
	}
	n := state.DeleteMember{
		ID:           id,
		Organization: this.organization.ID,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := this.core.db.Exec(ctx, "DELETE FROM members WHERE id = $1 AND organization = $2", id, this.organization.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("member %d does not exist", id)
		}
		return n, nil
	})
	return err
}

// HasMember reports whether the organization has a member with the given ID.
func (this *Organization) HasMember(id int) bool {
	return this.organization.HasMember(id)
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
	invitationToken, err := generateMemberToken()
	if err != nil {
		return err
	}
	if this.core.smtp == nil || this.core.memberEmailFrom == "" {
		return errors.Unprocessable(EmailSendFailed, "emails cannot be sent")
	}
	now := time.Now().UTC()
	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		exists, err := tx.QueryExists(ctx, "SELECT FROM members WHERE organization = $1 AND email = $2 AND invitation_token = ''", this.organization.ID, email)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.Unprocessable(MemberEmailExists, "a member with this email already exists")
		}
		_, err = tx.Exec(ctx, "INSERT INTO members (organization, name, email, password, avatar, invitation_token, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7) "+
			"ON CONFLICT (organization, email) DO UPDATE SET invitation_token = $6, created_at = $7",
			this.organization.ID, "", email, "", nil, invitationToken, now)
		return nil, err
	})
	if err != nil {
		return err
	}
	t := strings.ReplaceAll(emailTemplate, "${token}", html.EscapeString(invitationToken))
	emailToSend := &emailToSend{
		From:     this.core.memberEmailFrom,
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
	resetToken, err := generateMemberToken()
	if err != nil {
		return err
	}
	if this.core.smtp == nil || this.core.memberEmailFrom == "" {
		return errors.Unprocessable(EmailSendFailed, "emails cannot be sent")
	}
	now := time.Now().UTC()
	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		exists, err := tx.QueryExists(ctx, "SELECT FROM members WHERE organization = $1 AND email = $2 AND invitation_token = ''", this.organization.ID, email)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errResetPasswordTokenNotExist
		}
		_, err = tx.Exec(
			ctx,
			`UPDATE members SET reset_password_token = $1, reset_password_token_created_at = $2 WHERE organization = $3 AND email = $4`,
			resetToken,
			now,
			this.organization.ID,
			email,
		)
		return nil, err
	})
	if err != nil {
		if err == errResetPasswordTokenNotExist {
			// Do not return errors so that non-logged in users cannot
			// tell if the email exists or not.
			return nil
		}
		return err
	}
	t := strings.ReplaceAll(emailTemplate, "${token}", html.EscapeString(resetToken))
	emailToSend := &emailToSend{
		From:     this.core.memberEmailFrom,
		Subject:  "Your Meergo password reset",
		To:       email,
		BodyHTML: []byte(t),
	}
	err = sendMail(emailToSend, this.core.smtp)
	return err
}

// TestWorkspaceCreation tests a workspace creation. It tests that a warehouse
// with the provided driver name, settings and MCP settings (which can be nil)
// can be initialized.
//
// It returns an errors.UnprocessableError error with code:
//
//   - InvalidWarehouseSettings, if the warehouse settings are not valid.
//   - NotReadOnlyMCPSettings, if the MCP settings do not grant access to a
//     read-only user on the data warehouse.
//   - WarehouseDriverNotExist, if a warehouse driver does not exist.
//   - WarehouseNonInitializable, if the warehouse intended for connection is
//     not initializable.
func (this *Organization) TestWorkspaceCreation(ctx context.Context, name string,
	userSchema types.Type, uiPreferences UIPreferences, whName string,
	whSettings, whMCPSettings []byte, mode WarehouseMode) error {
	this.core.mustBeOpen()
	_, _, err := this.validateWorkspaceCreation(ctx, name, userSchema, uiPreferences,
		whName, whSettings, whMCPSettings, mode)

	return err
}

// UpdateAccessKey updates the name of the access key for the organization with
// the specified identifier. name must be between 1 and 100 runes in length.
//
// If the access key does not exist for the organization, it returns an
// errors.NotFound error.
func (this *Organization) UpdateAccessKey(ctx context.Context, id int, name string) error {
	this.core.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return errors.BadRequest("identifier %d is not a valid access key identifier", id)
	}
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return errors.BadRequest("%s", err)
	}
	result, err := this.core.db.Exec(ctx, "UPDATE access_keys SET name = $1 WHERE id = $2 AND organization = $3", name, id, this.organization.ID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.NotFound("access key %d does not exist", id)
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
			return nil, errors.NotFound("member %d does not exist", id)
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
// It tests that a warehouse with the provided driver name, settings and MCP
// settings (which can be nil) can be initialized, and returns an error if the
// arguments are not valid.
//
// It returns an errors.UnprocessableError error with code:
//
//   - InvalidWarehouseSettings, if the warehouse settings are not valid.
//   - NotReadOnlyMCPSettings, if the MCP settings do not grant access to a
//     read-only user on the data warehouse.
//   - WarehouseDriverNotExist, if a warehouse driver does not exist.
//   - WarehouseNonInitializable, if the warehouse intended for connection is
//     not initializable.
func (this *Organization) validateWorkspaceCreation(ctx context.Context, name string,
	userSchema types.Type, uiPreferences UIPreferences,
	whName string, whSettings []byte, whMCPSettings []byte, whMode WarehouseMode) ([]byte, []byte, error) {

	// Validate the parameters.
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return nil, nil, errors.BadRequest("%s", err)
	}
	if !userSchema.Valid() {
		return nil, nil, errors.BadRequest("user schema is invalid")
	}
	if err := validateUIPreferences(uiPreferences); err != nil {
		return nil, nil, errors.BadRequest("%s", err)
	}
	if whName == "" {
		return nil, nil, errors.BadRequest("warehouse driver name is empty")
	}
	switch whMode {
	case Normal, Inspection, Maintenance:
	default:
		return nil, nil, errors.BadRequest("warehouse mode is not valid")
	}

	// Perform additional checks on the compliance of the user schema.
	if err := checkAllowedPropertyUserSchema(userSchema); err != nil {
		return nil, nil, errors.BadRequest("%s", err)
	}
	if err := datastore.CheckConflictingProperties("users", userSchema); err != nil {
		return nil, nil, errors.BadRequest("%s", err)
	}

	// Normalize the warehouse settings.
	settings, err := this.core.datastore.NormalizeWarehouseSettings(whName, whSettings)
	if err != nil {
		if err == datastore.ErrWarehouseDriverNotExist {
			return nil, nil, errors.Unprocessable(WarehouseDriverNotExist, "warehouse driver %s does not exist", whName)
		}
		if err, ok := err.(*warehouses.SettingsError); ok {
			return nil, nil, errors.Unprocessable(InvalidWarehouseSettings, "data warehouse settings are not valid: %w", err.Err)
		}
		return nil, nil, err
	}

	// Normalize the warehouse MCP settings, if provided.
	if whMCPSettings != nil {
		// TODO(Gianluca): for https://github.com/meergo/meergo/issues/1833.
		if whName == "Snowflake" {
			return nil, nil, errors.BadRequest("MCP feature data is currently not supported for workspaces connected to a Snowflake warehouse")
		}
		var err error
		whMCPSettings, err = this.core.datastore.NormalizeWarehouseSettings(whName, whMCPSettings)
		if err != nil {
			if err, ok := err.(*warehouses.SettingsError); ok {
				return nil, nil, errors.Unprocessable(InvalidWarehouseSettings, "data warehouse MCP settings are not valid: %w", err.Err)
			}
			return nil, nil, err
		}
		if bytes.Equal(settings, whMCPSettings) {
			return nil, nil, errors.Unprocessable(InvalidWarehouseSettings, "the MCP settings must be different from the data warehouse settings")
		}
		err = this.core.datastore.CheckMCPSettings(ctx, whName, whMCPSettings)
		if err != nil {
			if err, ok := err.(*warehouses.SettingsNotReadOnly); ok {
				return nil, nil, errors.Unprocessable(NotReadOnlyMCPSettings, "invalid MCP settings: %s", err)
			}
			if err, ok := err.(*datastore.UnavailableError); ok {
				return nil, nil, errors.Unavailable("%s", err)
			}
			return nil, nil, err
		}
	}

	// Check if the warehouse is initializable.
	err = this.core.datastore.CanInitialize(ctx, whName, settings)
	if err != nil {
		if err, ok := err.(*warehouses.NonInitializableError); ok {
			return nil, nil, errors.Unprocessable(WarehouseNonInitializable, "data warehouse is not initializable: %w", err.Err)
		}
		if err, ok := err.(*datastore.UnavailableError); ok {
			return nil, nil, errors.Unavailable("%s", err)
		}
		return nil, nil, err
	}

	return settings, whMCPSettings, nil
}

// generateAccessKeyToken generates a new access key token.
func generateAccessKeyToken() string {
	// ⌈log₆₂ 2²⁵⁶⌉ ≈ 43 chars
	const base62alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	src := make([]byte, 43)
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
func validateMemberToSet(member MemberToSet, validateName bool, validateEmail bool, validatePassword bool) error {
	// Validate name.
	if validateName {
		if err := util.ValidateStringField("name", member.Name, 45); err != nil {
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
