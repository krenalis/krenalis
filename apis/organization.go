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
	"database/sql"
	"math"
	"math/big"
	"regexp"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/state"
)

var AuthenticationFailed errors.Code = "AuthenticationFailed"
var MemberEmailAlreadyExists errors.Code = "MemberEmailAlreadyExists"

var emailRegExp = regexp.MustCompile(`^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`)

// Organization represents an organization.
type Organization struct {
	apis         *APIs
	organization *state.Organization
	ID           int
	Name         string
}

type Avatar struct {
	Image    []byte // Image, in range [1, 200000]
	MimeType string // Mime type, must be `image/jpeg` or `image/png`
}

// Member represents a member of an organization.
type Member struct {
	ID     int
	Name   string
	Avatar *Avatar
	Email  string
}

// MemberToSet represents a member to set with the SetMember method.
type MemberToSet struct {
	Name     string // Name, in range [1, 60]
	Avatar   *Avatar
	Email    string // Email, in range [4,120] and must match `^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`
	Password string // Password, at least 8 characters long
}

// AddMember adds a new member to the organization. If a member with the same
// email already exists it returns an UnprocessableError with code
// MemberEmailAlreadyExists.
func (this *Organization) AddMember(ctx context.Context, member MemberToSet) error {
	this.apis.mustBeOpen()
	err := validateMemberToSet(member, true)
	if err != nil {
		return err
	}
	password, err := bcrypt.GenerateFromPassword([]byte(member.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		err := this.apis.db.QueryVoid(ctx, "SELECT FROM members WHERE email = $1 AND organization = $2", member.Email, this.organization.ID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil {
			return errors.Unprocessable(MemberEmailAlreadyExists, "a member with this email already exists")
		}
		if member.Avatar != nil {
			_, err = this.apis.db.Exec(ctx, "INSERT INTO members (organization, name, avatar.image, avatar.mime_type, email, password) VALUES ($1, $2, $3, $4, $5, $6)",
				this.organization.ID, member.Name, member.Avatar.Image, member.Avatar.MimeType, member.Email, string(password))
			return err
		}
		_, err = this.apis.db.Exec(ctx, "INSERT INTO members (organization, name, avatar, email, password) VALUES ($1, $2, $3, $4, $5)",
			this.organization.ID, member.Name, nil, member.Email, string(password))
		return err
	})
	return err
}

// AddWorkspace adds a workspace with the given name and privacy region, and
// returns its identifier. name must be between 1 and 100 runes long.
//
// It returns an errors.NotFoundError error if the organization does not exist
// anymore.
func (this *Organization) AddWorkspace(ctx context.Context, name string, region PrivacyRegion) (int, error) {

	this.apis.mustBeOpen()

	if name == "" || utf8.RuneCountInString(name) > 100 {
		return 0, errors.BadRequest("name %q is not valid", name)
	}
	switch region {
	case PrivacyRegionNotSpecified, PrivacyRegionEurope:
	default:
		return 0, errors.BadRequest("privacy region is not valid")
	}

	n := state.AddWorkspace{
		Organization:  this.organization.ID,
		Name:          name,
		PrivacyRegion: state.PrivacyRegion(region),
	}

	// Generate the identifier.
	var err error
	n.ID, err = generateRandomID()
	if err != nil {
		return 0, err
	}

	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO workspaces (id, organization, name, privacy_region) VALUES ($1, $2, $3, $4)",
			n.ID, n.Organization, n.Name, n.PrivacyRegion)
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
// If the member to delete does not exist, it returns an UnprocessableError with
// code MemberNotExist.
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
		return errors.Unprocessable(MemberNotExist, "member %d does not exist", id)
	}
	return nil
}

// Member returns the organization's member with identifier id.
func (this *Organization) Member(ctx context.Context, id int) (*Member, error) {
	this.apis.mustBeOpen()
	if id < 0 || id > math.MaxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid member identifier", id)
	}
	var member Member
	var avatarImage []byte
	var avatarMimeType *string
	err := this.apis.db.QueryRow(ctx, "SELECT id, name, (avatar).image, (avatar).mime_type, email FROM members WHERE id = $1 AND organization = $2", id, this.organization.ID).Scan(
		&member.ID, &member.Name, &avatarImage, &avatarMimeType, &member.Email)
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
	return &member, nil
}

// Members returns the organization's members sorted by name.
func (this *Organization) Members(ctx context.Context) ([]*Member, error) {
	this.apis.mustBeOpen()
	members := []*Member{}
	err := this.apis.db.QueryScan(ctx, "SELECT id, name, (avatar).image, (avatar).mime_type, email FROM members WHERE organization = $1 ORDER BY name", this.organization.ID, func(rows *postgres.Rows) error {
		var err error
		for rows.Next() {
			var member Member
			var avatarImage []byte
			var avatarMimeType *string
			if err = rows.Scan(&member.ID, &member.Name, &avatarImage, &avatarMimeType, &member.Email); err != nil {
				return err
			}
			if len(avatarImage) > 0 {
				member.Avatar = &Avatar{
					Image:    avatarImage,
					MimeType: *avatarMimeType,
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
// is empty, it does not change the password. If the member to set has an email
// that is already used by another member, it returns an UnprocessableError with
// code MemberEmailAlreadyExists.
func (this *Organization) SetMember(ctx context.Context, id int, member MemberToSet) error {
	this.apis.mustBeOpen()
	if id < 0 || id > math.MaxInt32 {
		return errors.BadRequest("identifier %d is not a valid member identifier", id)
	}
	err := validateMemberToSet(member, false)
	if err != nil {
		return err
	}
	var password []byte
	if member.Password != "" {
		password, err = bcrypt.GenerateFromPassword([]byte(member.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
	}
	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		err := this.apis.db.QueryVoid(ctx, "SELECT FROM members WHERE id <> $1 AND email = $2 AND organization = $3", id, member.Email, this.organization.ID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err != sql.ErrNoRows {
			return errors.Unprocessable(MemberEmailAlreadyExists, "a member with this email already exists")
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
		apis:                 this.apis,
		organization:         this,
		store:                this.apis.datastore.Store(id),
		workspace:            ws,
		ID:                   ws.ID,
		Name:                 ws.Name,
		Identifiers:          ws.Identifiers,
		AnonymousIdentifiers: AnonymousIdentifiers(ws.AnonymousIdentifiers),
		PrivacyRegion:        PrivacyRegion(ws.PrivacyRegion),
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
			apis:                 this.apis,
			organization:         this,
			store:                this.apis.datastore.Store(ws.ID),
			workspace:            ws,
			ID:                   ws.ID,
			Name:                 ws.Name,
			Identifiers:          ws.Identifiers,
			AnonymousIdentifiers: AnonymousIdentifiers(ws.AnonymousIdentifiers),
			PrivacyRegion:        PrivacyRegion(ws.PrivacyRegion),
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
func validateMemberToSet(member MemberToSet, isPasswordRequired bool) error {
	// Validate name.
	if member.Name == "" {
		return errors.BadRequest("name is empty")
	}
	if !utf8.ValidString(member.Name) {
		return errors.BadRequest("name is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(member.Name); n > 45 {
		return errors.BadRequest("name is longer than 45 runes")
	}
	// Validate avatar.
	if member.Avatar != nil {
		if member.Avatar.MimeType != "image/jpeg" && member.Avatar.MimeType != "image/png" {
			return errors.BadRequest("image must be in jpeg or png format")
		}
		if len(member.Avatar.Image) > 200*1024 {
			return errors.BadRequest("image is bigger than 200kb")
		}
	}
	// Validate email.
	if member.Email == "" {
		return errors.BadRequest("email is empty")
	}
	if !utf8.ValidString(member.Email) {
		return errors.BadRequest("email is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(member.Email); n > 120 {
		return errors.BadRequest("email is longer than 120 runes")
	}
	if !emailRegExp.MatchString(member.Email) {
		return errors.BadRequest("email is not a valid email address")
	}
	// Validate password.
	if isPasswordRequired && member.Password == "" {
		return errors.BadRequest("password is empty")
	}
	if member.Password != "" {
		if !utf8.ValidString(member.Password) {
			return errors.BadRequest("password is not UTF-8 encoded")
		}
		if n := utf8.RuneCountInString(member.Password); n < 8 {
			return errors.BadRequest("password must be at least 8 characters long")
		}
		if n := utf8.RuneCountInString(member.Password); n > 72 {
			return errors.BadRequest("password is longer than 72 runes")
		}
	}
	return nil
}
