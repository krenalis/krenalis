//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"database/sql"
	"errors"
	"regexp"
	"sort"
	"sync"

	"chichi/apis/postgres"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"golang.org/x/crypto/bcrypt"
)

type Accounts struct {
	*APIs
	state accountsState
}

type accountsState struct {
	sync.Mutex
	ids map[int]*Account
}

var errAccountNotFound = errors.New("account does not exist")

// get returns the account with identifier id.
// Returns the errAccountNotFound error if the account does not exist.
func (this *Accounts) get(id int) (*Account, error) {
	this.state.Lock()
	c, ok := this.state.ids[id]
	this.state.Unlock()
	if ok {
		return c, nil
	}
	return nil, errAccountNotFound
}

// newAccounts returns a new *Accounts value.
func newAccounts(apis *APIs, accounts map[int]*Account) *Accounts {
	return &Accounts{APIs: apis, state: accountsState{ids: accounts}}
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
// Returns an error is the account does not exist.
func (this *Accounts) As(id int) (*Account, error) {
	return this.get(id)
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
		if err == sql.ErrNoRows {
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
	return this.count()
}

// count returns the total number of accounts.
func (this *Accounts) count() int {
	this.state.Lock()
	l := len(this.state.ids)
	this.state.Unlock()
	return l
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
	acc, err := this.get(id)
	if err != nil {
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

type AccountSort int

const (
	SortByName AccountSort = iota
	SortByEmail
)

// list returns all accounts.
func (this *Accounts) list() []*Account {
	this.state.Lock()
	accounts := make([]*Account, len(this.state.ids))
	i := 0
	for _, account := range this.state.ids {
		accounts[i] = account
	}
	this.state.Unlock()
	return accounts
}

// List returns a list of *AccountInfo, in the given order, describing all
// accounts but starting from first and up to limit. first must be >= 0 and
// limit must be > 0.
func (this *Accounts) List(order AccountSort, first, limit int) []*AccountInfo {
	if order != SortByName && order != SortByEmail {
		panic("invalid order")
	}
	if limit <= 0 {
		panic("invalid limit")
	}
	if first < 0 {
		panic("invalid first")
	}
	accounts := this.list()
	count := len(accounts)
	if first >= count {
		return []*AccountInfo{}
	}
	if first+limit > count {
		limit = count - first
	}
	sort.Slice(accounts, func(i, j int) bool {
		a, b := accounts[i], accounts[j]
		switch order {
		case SortByName:
			return a.name < b.name || a.name == b.name && a.id < b.id
		case SortByEmail:
			return a.email < b.email || a.email == b.email && a.id < b.id
		}
		return false
	})
	accounts = accounts[first : first+limit]
	infos := make([]*AccountInfo, len(accounts))
	for i, account := range accounts {
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
