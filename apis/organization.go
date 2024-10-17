//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"html"
	"maps"
	"math"
	"math/big"
	"net/smtp"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/apis/datastore"
	"github.com/meergo/meergo/apis/errors"
	"github.com/meergo/meergo/apis/postgres"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"

	"github.com/jordan-wright/email"
	"golang.org/x/crypto/bcrypt"
)

var emailRegExp = regexp.MustCompile(`^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`)

// invitationTokenMaxAge represents the max age of an invitation token (3 days).
const invitationTokenMaxAge = 3 * 24 * 60 * 60

// Organization represents an organization.
type Organization struct {
	apis         *APIs
	organization *state.Organization
	ID           int
	Name         string
}

// Member represents a member of an organization.
type Member struct {
	ID         int
	Name       string
	Email      string
	Avatar     *Avatar
	Invitation string // If the member has been invited, it is "Invited or "Expired"
	CreatedAt  time.Time
}

// Avatar represents an avatar of a member.
type Avatar struct {
	Image    []byte // Image, in range [1, 200000]
	MimeType string // Mime type, must be `image/jpeg` or `image/png`
}

// MemberToSet represents a member to set with the SetMember method.
type MemberToSet struct {
	Name     string // Name, in range [1, 60]
	Email    string // Email, in range [4,120] and must match `^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`
	Password string // Password, at least 8 characters long
	Avatar   *Avatar
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

// CanInitializeWarehouse indicates whether the warehouse with the provided name
// and settings can be initialized.
//
// It returns an errors.UnprocessableError error with code:
//
//   - DataWarehouseFailed, if an operation on the data warehouse fails;
//   - DataWarehouseNotExist, if a data warehouse with the provided name does
//     not exist;
//   - InvalidWarehouseSettings, if the warehouse settings are not valid;
//   - WarehouseNotInitializable, if the warehouse intended for connection is
//     not initializable.
func (this *Organization) CanInitializeWarehouse(ctx context.Context, name string, settings []byte) error {
	this.apis.mustBeOpen()

	// Validate the parameters.
	if name == "" {
		return errors.BadRequest("warehouse name is empty")
	}

	// Normalize the warehouse settings.
	settings, err := this.apis.datastore.NormalizeWarehouseSettings(name, settings)
	if err != nil {
		if err == datastore.DataWarehouseNotExist {
			return errors.Unprocessable(DataWarehouseNotExist, "data warehouse %q does not exist", name)
		}
		if err, ok := err.(*datastore.SettingsError); ok {
			return errors.Unprocessable(InvalidWarehouseSettings, "data warehouse settings are not valid: %w", err.Err)
		}
		return err
	}

	// Check if the warehouse is initializable.
	err = this.apis.datastore.CanInitialize(ctx, name, settings)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseNotInitializableError); ok {
			return errors.Unprocessable(WarehouseNotInitializable, "%w", err)
		}
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			return errors.Unprocessable(DataWarehouseFailed, "data warehouse error: %w", err)
		}
		return err
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
	this.apis.mustBeOpen()
	err := validateMemberEmail(email)
	if err != nil {
		return errors.BadRequest("%s", err)
	}
	invitationToken, err := generateInvitationToken()
	if err != nil {
		return err
	}
	if this.apis.smtp == nil {
		return errors.Unprocessable(EmailSendFailed, "emails cannot be sent")
	}
	now := time.Now().UTC()
	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		err := this.apis.db.QueryVoid(ctx, "SELECT FROM members WHERE organization = $1 AND email = $2 AND invitation_token = ''", this.organization.ID, email)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil {
			return errors.Unprocessable(MemberEmailExists, "a member with this email already exists")
		}
		_, err = this.apis.db.Exec(ctx, "INSERT INTO members (organization, name, email, password, avatar, invitation_token, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7) "+
			"ON CONFLICT (organization, email) DO UPDATE SET invitation_token = $6, created_at = $7",
			this.organization.ID, "", email, "", nil, invitationToken, now)
		return err
	})
	if err != nil {
		return err
	}
	t := strings.ReplaceAll(emailTemplate, "${token}", html.EscapeString(invitationToken))
	emailToSend := &emailToSend{
		From:     this.apis.smtp.User,
		Subject:  "You have been invited to Meergo",
		To:       email,
		BodyHTML: []byte(t),
	}
	err = sendMail(emailToSend, this.apis.smtp)
	return err
}

// defaultUserSchema is the default user schema (without meta properties).
// It must be kept in sync with the SQL script that initializes the data
// warehouse.
var defaultUserSchema = types.Object([]types.Property{
	{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
})

// AddWorkspace adds a workspace with the given name and privacy region, and
// connects to a data warehouse of the provided name and settings. Returns the
// identifier of the workspace that has been created. name must be between 1 and
// 100 runes long.
//
// whMode specifies the initial mode of the workspace's data warehouse.
//
// It returns an errors.NotFoundError error if the organization does not exist
// anymore.
//
// It returns an errors.UnprocessableError error with code:
//
//   - DataWarehouseFailed, if an operation on the data warehouse fails;
//   - DataWarehouseNotExist, if a data warehouse with the provided name does
//     not exist;
//   - InvalidWarehouseSettings, if the warehouse settings are not valid;
func (this *Organization) AddWorkspace(ctx context.Context, name string, region PrivacyRegion, whName string, whSettings []byte, whMode WarehouseMode) (int, error) {

	this.apis.mustBeOpen()

	// Validate the parameters.
	if name == "" || utf8.RuneCountInString(name) > 100 || containsNUL(name) {
		return 0, errors.BadRequest("name %q is not valid", name)
	}
	switch region {
	case PrivacyRegionNotSpecified, PrivacyRegionEurope:
	default:
		return 0, errors.BadRequest("privacy region is not valid")
	}
	if whName == "" {
		return 0, errors.BadRequest("warehouse name is empty")
	}

	// Normalize the warehouse settings.
	whSettings, err := this.apis.datastore.NormalizeWarehouseSettings(whName, whSettings)
	if err != nil {
		if err == datastore.DataWarehouseNotExist {
			return 0, errors.Unprocessable(DataWarehouseNotExist, "data warehouse %q does not exist", name)
		}
		if err, ok := err.(*datastore.SettingsError); ok {
			return 0, errors.Unprocessable(InvalidWarehouseSettings, "data warehouse settings are not valid: %w", err.Err)
		}
		return 0, err
	}

	// Check if the warehouse is initializable.
	err = this.apis.datastore.CanInitialize(ctx, whName, whSettings)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseNotInitializableError); ok {
			return 0, errors.Unprocessable(WarehouseNotInitializable, "data warehouse cannot be initialized: %w", err)
		}
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			return 0, errors.Unprocessable(DataWarehouseFailed, "data warehouse error: %w", err)
		}
		return 0, err
	}

	// Initialize the data warehouse.
	err = this.apis.datastore.Initialize(ctx, whName, whSettings)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			return 0, errors.Unprocessable(DataWarehouseFailed, "cannot check the data warehouse: %w", err)
		}
		return 0, err
	}

	n := state.AddWorkspace{
		Organization:                   this.organization.ID,
		Name:                           name,
		UserSchema:                     defaultUserSchema,
		ResolveIdentitiesOnBatchImport: true,
		PrivacyRegion:                  state.PrivacyRegion(region),
		Warehouse: state.Warehouse{
			Name:     whName,
			Mode:     state.WarehouseMode(whMode),
			Settings: whSettings,
		},
	}

	// Generate the identifier.
	n.ID, err = generateRandomID()
	if err != nil {
		return 0, err
	}

	// Encode the user schema to JSON.
	userSchema, err := json.Marshal(n.UserSchema)
	if err != nil {
		return 0, err
	}

	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO workspaces (id, organization, name,"+
			" user_schema, resolve_identities_on_batch_import, privacy_region,"+
			" warehouse_name, warehouse_mode, warehouse_settings)"+
			" VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
			n.ID, n.Organization, n.Name, userSchema, n.ResolveIdentitiesOnBatchImport,
			n.PrivacyRegion, n.Warehouse.Name, n.Warehouse.Mode, n.Warehouse.Settings)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "workspaces_keys_organization_fkey" {
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

// AuthenticateMember authenticates a member of the organization given its email
// and password. email's length must be in range [4, 120] and must be a valid
// email address. password's length must be at least 8 character long.
//
// If a member with the provided email does not exist or the password does not
// correspond, it returns an errors.UnprocessableError error with code
// AuthenticationFailed.
func (this *Organization) AuthenticateMember(ctx context.Context, email, password string) (int, error) {

	this.apis.mustBeOpen()

	// Validate email.
	if email == "" {
		return 0, errors.BadRequest("email is empty")
	}
	if !utf8.ValidString(email) {
		return 0, errors.BadRequest("email is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(email); n > 120 {
		return 0, errors.BadRequest("email is longer than 120 runes")
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
	err := this.apis.db.QueryRow(ctx, "SELECT id, password FROM members WHERE organization = $1 AND email = $2", this.organization.ID, email).Scan(&id, &hashedPassword)
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

// DeleteMember deletes a member of the organization with identifier id.
// If the member does not exist, it returns an errors.NotFound error.
func (this *Organization) DeleteMember(ctx context.Context, id int) error {
	this.apis.mustBeOpen()
	if id < 0 || id > math.MaxInt32 {
		return errors.BadRequest("identifier %d is not a valid member identifier", id)
	}
	result, err := this.apis.db.Exec(ctx, "DELETE FROM members WHERE id = $1 AND organization = $2", id, this.organization.ID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.NotFound("member %d does not exist", id)
	}
	return nil
}

// Member returns the organization's member with identifier id.
// If the member does not exist, it returns an errors.NotFound error.
func (this *Organization) Member(ctx context.Context, id int) (*Member, error) {
	this.apis.mustBeOpen()
	if id < 0 || id > math.MaxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid member identifier", id)
	}
	var member Member
	var avatarImage []byte
	var avatarMimeType *string
	var invitationToken string
	err := this.apis.db.QueryRow(ctx, "SELECT id, name, email, (avatar).image, (avatar).mime_type, invitation_token, created_at FROM members WHERE id = $1 AND organization = $2", id, this.organization.ID).Scan(
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
	this.apis.mustBeOpen()
	members := []*Member{}
	err := this.apis.db.QueryScan(ctx, "SELECT id, name, email, (avatar).image, (avatar).mime_type, invitation_token, created_at FROM members WHERE organization = $1 ORDER BY name", this.organization.ID, func(rows *postgres.Rows) error {
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

// SetMember sets a member of the organization with identifier id. If password
// is empty, it does not change the password.
//
// If the member does not exist, it returns an errors.NotFound error. If the
// member to set has an email that is already used by another member, it returns
// an errors.UnprocessableError error with code MemberEmailExists.
func (this *Organization) SetMember(ctx context.Context, id int, member MemberToSet) error {
	this.apis.mustBeOpen()
	if id < 0 || id > math.MaxInt32 {
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
	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		err := this.apis.db.QueryVoid(ctx, "SELECT FROM members WHERE id = $1 AND organization = $2", id, this.organization.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.NotFound("member %d does not exist", id)
			}
			return err
		}
		err = this.apis.db.QueryVoid(ctx, "SELECT FROM members WHERE id <> $1 AND organization = $2 AND email = $3", id, this.organization.ID, member.Email)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err != sql.ErrNoRows {
			return errors.Unprocessable(MemberEmailExists, "a member with this email already exists")
		}
		_, err = this.apis.db.Exec(ctx, "UPDATE members SET name = $1, email = $2 WHERE id = $3 AND organization = $4",
			member.Name, member.Email, id, this.organization.ID)
		if err != nil {
			return err
		}
		if member.Avatar != nil {
			_, err = this.apis.db.Exec(ctx, "UPDATE members SET avatar.image = $1, avatar.mime_type = $2 WHERE id = $3 AND organization = $4",
				member.Avatar.Image, member.Avatar.MimeType, id, this.organization.ID)
		} else {
			_, err = this.apis.db.Exec(ctx, "UPDATE members SET avatar = $1 WHERE id = $2 AND organization = $3",
				nil, id, this.organization.ID)
		}
		if err != nil {
			return err
		}
		if password != nil {
			_, err = this.apis.db.Exec(ctx, "UPDATE members SET password = $1 WHERE id = $2 AND organization = $3",
				string(password), id, this.organization.ID)
		}
		return err
	})
	return err
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

// Workspace returns the organization's workspace with identifier id.
//
// It returns an errors.NotFound error if the workspace does not exist.
func (this *Organization) Workspace(id int) (*Workspace, error) {
	this.apis.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid workspace identifier", id)
	}
	ws, ok := this.organization.Workspace(id)
	if !ok {
		return nil, errors.NotFound("workspace %d does not exist", id)
	}
	workspace := Workspace{
		apis:                           this.apis,
		organization:                   this,
		store:                          this.apis.datastore.Store(id),
		workspace:                      ws,
		ID:                             ws.ID,
		Name:                           ws.Name,
		UserSchema:                     ws.UserSchema,
		UserPrimarySources:             maps.Clone(ws.UserPrimarySources),
		ResolveIdentitiesOnBatchImport: ws.ResolveIdentitiesOnBatchImport,
		Identifiers:                    ws.Identifiers,
		WarehouseMode:                  WarehouseMode(ws.Warehouse.Mode),
		PrivacyRegion:                  PrivacyRegion(ws.PrivacyRegion),
		DisplayedProperties:            DisplayedProperties(ws.DisplayedProperties),
	}
	return &workspace, nil
}

// Workspaces returns the workspaces of the organization.
func (this *Organization) Workspaces() []*Workspace {
	this.apis.mustBeOpen()
	workspaces := this.organization.Workspaces()
	infos := make([]*Workspace, len(workspaces))
	for i, ws := range workspaces {
		workspace := Workspace{
			apis:                           this.apis,
			organization:                   this,
			store:                          this.apis.datastore.Store(ws.ID),
			workspace:                      ws,
			ID:                             ws.ID,
			Name:                           ws.Name,
			UserSchema:                     ws.UserSchema,
			UserPrimarySources:             maps.Clone(ws.UserPrimarySources),
			ResolveIdentitiesOnBatchImport: ws.ResolveIdentitiesOnBatchImport,
			Identifiers:                    ws.Identifiers,
			WarehouseMode:                  WarehouseMode(ws.Warehouse.Mode),
			PrivacyRegion:                  PrivacyRegion(ws.PrivacyRegion),
			DisplayedProperties:            DisplayedProperties(ws.DisplayedProperties),
		}
		infos[i] = &workspace
	}
	return infos
}

var bigMaxInt32 = big.NewInt(math.MaxInt32)

// generateRandomID generates a random identifier in [1, maxInt32].
func generateRandomID() (int, error) {
	n, err := rand.Int(rand.Reader, bigMaxInt32)
	if err != nil {
		return 0, err
	}
	return int(n.Int64()) + 1, nil
}

// validateMemberToSet validates a member to add or set and returns an error
// if the member is not valid.
func validateMemberToSet(member MemberToSet, validateEmail bool, validatePassword bool) error {
	// Validate name.
	if member.Name == "" {
		return errors.New("name is empty")
	}
	if containsNUL(member.Name) {
		return errors.New("name contains NUL rune")
	}
	if !utf8.ValidString(member.Name) {
		return errors.New("name is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(member.Name); n > 45 {
		return errors.New("name is longer than 45 runes")
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

// validateMemberEmail validates a member's email and returns an error if it is
// not valid.
func validateMemberEmail(email string) error {
	if email == "" {
		return errors.New("email is empty")
	}
	if !utf8.ValidString(email) {
		return errors.New("email is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(email); n > 120 {
		return errors.New("email is longer than 120 runes")
	}
	if !emailRegExp.MatchString(email) {
		return errors.New("email is not a valid email address")
	}
	return nil
}

// generateInvitationToken generates a token.
func generateInvitationToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// isInvitationTokenExpired checks if the invitation token of a member is expired, given
// the member's creation time.
func isInvitationTokenExpired(createdAt time.Time) bool {
	tokenExpiration := createdAt.Add(time.Duration(invitationTokenMaxAge) * time.Second)
	now := time.Now()
	return now.After(tokenExpiration)
}
