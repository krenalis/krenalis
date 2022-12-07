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
	"strings"

	"chichi/pkg/open2b/sql"

	"golang.org/x/crypto/bcrypt"
)

type Accounts struct {
	*APIs
}

// Account represents an account.
type Account struct {
	ID          int
	Name        string
	Email       string
	Properties  []int
	InternalIPs []string
}

var emailRegExp = regexp.MustCompile(`^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`)

var ErrEmailInvalid = errors.New("email is not valid")
var ErrPasswordInvalid = errors.New("password is not valid")
var ErrAuthenticationFailed = errors.New("authentication failed")

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
	err := this.myDB.QueryRow("SELECT `id`, `password`\nFROM `accounts`\nWHERE `email` = ?", email).Scan(&id, &hashedPassword)
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
func (this *Accounts) Count() (int, error) {
	var count int
	err := this.myDB.QueryRow("SELECT COUNT(*)\nFROM `accounts`").Scan(&count)
	return count, err
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
	result, err := this.myDB.Exec("INSERT INTO `accounts` SET `email` = ?, `password` = ?", email, string(hashedPassword))
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return int(id), err
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
	_, err := this.myDB.Exec("DELETE `c`, `p`, `d`\n" +
		"FROM `accounts` AS `a`\n" +
		"LEFT JOIN `proprieties` AS `p` ON `p`.`account` = `a`.`id`\n" +
		"LEFT JOIN `domains` AS `d` ON `d`.`property` = `p`.`id`")
	return err
}

// Find returns the accounts in the given order limited by limit and first.
func (this *Accounts) Find(order string, limit, first int) ([]*Account, error) {
	if order != "name" && order != "email" {
		panic("apis: invalid accounts order")
	}
	if limit < 0 || first < 0 {
		panic("apis: invalid accounts limit or first")
	}
	stmt := "SELECT `id`, `name`, `email`\nFROM `accounts`"
	if order != "" {
		stmt += "\nORDER BY `" + order + "`"
	}
	stmt += sql.LimitFirstStatement(limit, first)
	accounts := make([]*Account, 0, 0)
	err := this.myDB.QueryScan(stmt, func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var account Account
			if err = rows.Scan(&account.ID, &account.Name, &account.Email); err != nil {
				return err
			}
			accounts = append(accounts, &account)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// Get returns the account with identifier id. If it does not exist, returns nil.
func (this *Accounts) Get(id int) (*Account, error) {
	if id < 1 {
		panic("apis: invalid account identifier")
	}
	account := Account{ID: id}
	var ips string
	err := this.myDB.QueryRow("SELECT `name`, `email`, `internal_ips`\nFROM `accounts`\nWHERE `id` = ?", id).Scan(&account.Name, &account.Email, &ips)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	account.InternalIPs = strings.Fields(ips)
	return &account, nil
}
