//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"errors"
	"regexp"

	"chichi/pkg/open2b/sql"

	"golang.org/x/crypto/bcrypt"
)

type Customers struct {
	*APIs
}

// Customer represents a customer.
type Customer struct {
	ID    int
	Name  string
	Email string
}

var emailRegExp = regexp.MustCompile(`^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`)

var ErrEmailInvalid = errors.New("email is not valid")
var ErrPasswordInvalid = errors.New("password is not valid")
var ErrAuthenticationFailed = errors.New("authentication failed")

// Authenticate authenticates a customer given its email and password. If the
// authentication fails, it returns the ErrAuthenticationFailed error.
func (this *Customers) Authenticate(email, password string) (int, error) {
	if !emailRegExp.MatchString(email) {
		return 0, ErrEmailInvalid
	}
	if len(password) < 8 {
		return 0, ErrPasswordInvalid
	}
	var id int
	var hashedPassword []byte
	err := this.myDB.QueryRow("SELECT `id`, `password`\nFROM `customers`\nWHERE `email` = ?").Scan(&id, &hashedPassword)
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

// Count returns the total number of customers.
func (this *Customers) Count() (int, error) {
	var count int
	err := this.myDB.QueryRow("SELECT COUNT(*)\nFROM `customers`").Scan(&count)
	return count, err
}

// Create a new customer given its email and password and returns its
// identifier. If the email is not valid it panics with error
// ErrEmailInvalid and if the password is not valid it panics with
// error ErrPasswordInvalid.
func (this *Customers) Create(email, password string) (int, error) {
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
	result, err := this.myDB.Exec("INSERT INTO `customers` SET `email` = ?, `password` = ?", email, string(hashedPassword))
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return int(id), err
}

// Delete deletes the customers with the given identifiers.
func (this *Customers) Delete(ids []int) error {
	if len(ids) == 0 {
		panic("apis: empty customers to delete")
	}
	for _, id := range ids {
		if id < 1 {
			panic("apis: invalid customer identifier to delete")
		}
	}
	_, err := this.myDB.Exec("DELETE FROM `customers` WHERE `id` IN " + sql.Quote(ids))
	return err
}

// Find returns the customers in the given order limited by limit and first.
func (this *Customers) Find(order string, limit, first int) ([]*Customer, error) {
	if order != "name" && order != "email" {
		panic("apis: invalid customers order")
	}
	if limit < 0 || first < 0 {
		panic("apis: invalid customers limit or first")
	}
	stmt := "SELECT `id`, `name`, `email`\nFROM `customers`"
	if order != "" {
		stmt += "\nORDER BY `" + order + "`"
	}
	stmt += sql.LimitFirstStatement(limit, first)
	customers := make([]*Customer, 0, 0)
	err := this.myDB.QueryScan(stmt, func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var customer Customer
			if err = rows.Scan(&customer.ID, &customer.Name, &customer.Email); err != nil {
				return err
			}
			customers = append(customers, &customer)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return customers, nil
}

// Get returns the customer with identifier id. If it does not exist, returns nil.
func (this *Customers) Get(id int) (*Customer, error) {
	if id < 1 {
		panic("apis: invalid customer identifier")
	}
	customer := Customer{ID: id}
	err := this.myDB.QueryRow("SELECT `name`, `email`\nFROM `customers`\nWHERE `id` = ?", id).Scan(&customer.Name, &customer.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}
