//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"errors"
	"regexp"
	"sort"

	"chichi/apis/postgres"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"golang.org/x/crypto/bcrypt"
)

type Accounts struct {
	*APIs
	accounts map[int]*Account
}

// newAccounts returns a new *Accounts value.
func newAccounts(apis *APIs, accounts map[int]*Account) *Accounts {
	return &Accounts{APIs: apis, accounts: accounts}
}

// Account represents an account.
type Account struct {
	apis        *APIs
	db          *postgres.DB
	chDB        chDriver.Conn
	Workspaces  *Workspaces
	id          int
	name        string
	email       string
	internalIPs []string
}

// An AccountInfo describes an account as returned by Get and List.
type AccountInfo struct {
	ID          int
	Name        string
	Email       string
	InternalIPs []string
}

var emailRegExp = regexp.MustCompile(`^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`)

var ErrEmailInvalid = errors.New("email is not valid")
var ErrPasswordInvalid = errors.New("password is not valid")
var ErrAuthenticationFailed = errors.New("authentication failed")

// As returns the account with identifier id.
// Returns an error is the workspace does not exist.
func (this *Accounts) As(id int) (*Account, error) {
	acc, ok := this.accounts[id]
	if !ok {
		return nil, errors.New("account does not exit")
	}
	return acc, nil
}

// Authenticate authenticates an account given its email and password. If the
// authentication fails, it returns the ErrAuthenticationFailed error.
func (this *Accounts) Authenticate(email, password string) (int, error) {
	if !emailRegExp.MatchString(email) {
		return 0, ErrEmailInvalid
	}
	if len(password) < 8 {
		return 0, ErrPasswordInvalid
	}
	var id int
	var hashedPassword []byte
	err := this.db.QueryRow("SELECT id, password FROM accounts WHERE email = $1", email).Scan(&id, &hashedPassword)
	if err != nil {
		if err == postgres.ErrNoRows {
			return 0, ErrAuthenticationFailed
		}
		return 0, err
	}
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		return 0, ErrAuthenticationFailed
	}
	return id, nil
}

// Count returns the total number of accounts.
func (this *Accounts) Count() int {
	return len(this.accounts)
}

// Create a new account given its email and password and returns its
// identifier. If the email is not valid it panics with error
// ErrEmailInvalid and if the password is not valid it panics with
// error ErrPasswordInvalid.
func (this *Accounts) Create(email, password string) (int, error) {
	if !emailRegExp.MatchString(email) {
		return 0, ErrEmailInvalid
	}
	if len(password) < 8 {
		return 0, ErrPasswordInvalid
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	var id int
	err = this.db.QueryRow("INSERT INTO accounts (email, password) VALUES ($1, $2)",
		email, string(hashedPassword)).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, err
}

// Delete deletes the accounts with the given identifiers.
func (this *Accounts) Delete(ids []int) error {
	if len(ids) == 0 {
		panic("apis: empty accounts to delete")
	}
	for _, id := range ids {
		if id < 1 {
			panic("apis: invalid account identifier to delete")
		}
	}
	panic("TO BE IMPLEMENTED")
	//_, err := this.db.Exec("DELETE `c`, `p`, `d`\n" +
	//	"FROM `accounts` AS `a`\n" +
	//	"LEFT JOIN `proprieties` AS `p` ON `p`.`account` = `a`.`id`\n" +
	//	"LEFT JOIN `domains` AS `d` ON `d`.`property` = `p`.`id`")
	//return err
}

// Get returns an AccountInfo describing the account with identifier id.
// Returns the ErrAccountNotFound if the account does not exist.
func (this *Accounts) Get(id int) (*AccountInfo, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.New("invalid account identifier")
	}
	acc, ok := this.accounts[id]
	if !ok {
		return nil, ErrAccountNotFound
	}
	ips := make([]string, len(acc.internalIPs))
	copy(ips, acc.internalIPs)
	info := AccountInfo{
		ID:          acc.id,
		Name:        acc.name,
		Email:       acc.email,
		InternalIPs: ips,
	}
	return &info, nil
}

// List returns a list AccountInfo, in the given order, describing all accounts
// but limited by limit and first.
// order can be "name" or "email". limit must be > 0 and first must be >= 0.
func (this *Accounts) List(order string, limit, first int) []*AccountInfo {
	if order != "name" && order != "email" {
		panic("invalid order")
	}
	if limit <= 0 {
		panic("invalid limit")
	}
	if first < 0 {
		panic("invalid first")
	}
	if first >= len(this.accounts) {
		return []*AccountInfo{}
	}
	if first+limit > len(this.accounts) {
		limit = len(this.accounts) - first
	}
	accounts := make([]*Account, 0, len(this.accounts))
	for _, account := range this.accounts {
		accounts = append(accounts, account)
	}
	if order == "name" {
		sort.Slice(accounts, func(i, j int) bool {
			a := accounts
			return a[i].name < a[j].name || a[i].name == a[j].name && a[i].id < a[j].id
		})
	} else {
		sort.Slice(accounts, func(i, j int) bool {
			a := accounts
			return a[i].email < a[j].email || a[i].email == a[j].email && a[i].id < a[j].id
		})
	}
	infos := make([]*AccountInfo, limit)
	for i, account := range accounts[first : first+limit] {
		ips := make([]string, len(account.internalIPs))
		copy(ips, account.internalIPs)
		infos[i] = &AccountInfo{
			ID:          account.id,
			Name:        account.name,
			Email:       account.email,
			InternalIPs: ips,
		}
	}
	return infos
}
