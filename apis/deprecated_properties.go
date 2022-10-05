//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"chichi/pkg/open2b/sql"

	"github.com/go-sql-driver/mysql"
)

type Properties struct {
	*API
	SmartEvents   *SmartEvents
	Visualization *Visualization
	id            int
}

type Property struct {
	ID      int
	Code    string
	Domains []string
}

var ErrCustomerNotFound = errors.New("customer does not exist")
var ErrPropertyNotFound = errors.New("property does not exist")
var ErrDomainNameNotValid = errors.New("domain name is not valid")

// Create creates a new property for the current customer and returns its
// identifier. If the customer does not exist anymore, it returns the
// ErrCustomerNotFound error.
func (this *Properties) Create() (int, error) {
	var id int
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		exists, err := tx.Table("Customers").Exists(sql.Where{"id": this.customer})
		if err != nil {
			return err
		}
		if !exists {
			return ErrCustomerNotFound
		}
		var tries = 0
		for tries < 10 {
			code, err := generatePropertyCode()
			if err != nil {
				return err
			}
			id, err = tx.Table("Properties").Add(map[string]any{"code": code, "customer": this.customer}, nil)
			if err != nil {
				if err2, ok := err.(*mysql.MySQLError); ok && err2.Number == 1062 {
					tries++
					continue
				}
				return err
			}
		}
		if tries == 10 {
			return errors.New("apis: cannot generate a property identifier")
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// AddDomain adds the domain named domain to the property with identifier id.
// It does nothing if the property already has the domain to add.
//
// If domain is not a valid domain name, it returns the ErrDomainNameNotValid
// error. If the property does not exist, or it does not belong to the current
// customer, it returns the ErrPropertyNotFound error.
func (this *Properties) AddDomain(id int, domain string) error {
	if id < 1 {
		panic("apis: invalid property identifier")
	}
	if !isValidDomainName(domain) {
		return ErrDomainNameNotValid
	}
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var customer int
		err := tx.QueryRow("SELECT `customer` FROM `properties` WHERE `id` = ?", id).Scan(&customer)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrPropertyNotFound
			}
			return err
		}
		if customer != this.customer {
			return ErrPropertyNotFound
		}
		_, err = tx.Table("Domains").Add(map[string]any{"property": id, "domain": domain}, sql.Ignore)
		return err
	})
	return err
}

// Delete deletes the properties with the given identifiers of the current
// customer. It does not return an error if the customer does not exist.
func (this *Properties) Delete(ids []int) error {
	if len(ids) == 0 {
		panic("apis: empty properties")
	}
	for _, id := range ids {
		if id < 1 {
			panic("apis: invalid property identifier")
		}
	}
	_, err := this.myDB.Exec("DELETE `p`, `d`\n"+
		"FROM `properties` AS `p` LEFT JOIN `domains` AS `d` ON `p`.`id` = `d`.`property`\n"+
		"WHERE `customer` = ? AND `p`.`id` IN "+sql.Quote(ids), this.customer)
	return err
}

// Find returns all the properties of the customer.
func (this *Properties) Find() ([]*Property, error) {
	properties := make([]*Property, 0, 0)
	stmt := "SELECT `id`, `code`, `domain`\nFROM `properties`\nLEFT JOIN `domains` ON `property` = `id`\nWHERE `customer` = ?\nORDER BY `id`, `domain`"
	err := this.myDB.QueryScan(stmt, func(rows *sql.Rows) error {
		var id int
		var code, domain string
	Rows:
		for rows.Next() {
			if err := rows.Scan(&id, &code, &domain); err != nil {
				return err
			}
			for _, property := range properties {
				if property.ID == id {
					property.Domains = append(property.Domains, domain)
					continue Rows
				}
			}
			property := &Property{ID: id, Code: code}
			if domain != "" {
				property.Domains = []string{domain}
			}
			properties = append(properties, property)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return properties, nil
}

// Get returns the property, of the current customer, with the given
// identifier. If the customer or the property do not exist, it returns nil.
func (this *Properties) Get(id int) (*Property, error) {
	if id < 1 {
		panic("apis: invalid property identifier")
	}
	property := Property{}
	stmt := "SELECT `code`, `domain`\nFROM `properties`\nLEFT JOIN `domains` ON `property` = `id`\nWHERE `customer` = ? AND `id` = ?"
	err := this.myDB.QueryScan(stmt, func(rows *sql.Rows) error {
		var code, domain string
		for rows.Next() {
			if err := rows.Scan(&code, &domain); err != nil {
				return err
			}
			property.ID = id
			property.Code = code
			if domain == "" {
				property.Domains = []string{}
			} else {
				property.Code = code
				if property.Domains == nil {
					property.Domains = []string{domain}
				} else {
					property.Domains = append(property.Domains)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if property.ID == 0 {
		return nil, nil
	}
	return &property, nil
}

// RemoveDomain removes the domain named domain from the property with
// identifier id. It does nothing if the property does not have the domain to
// be removed.
//
// If domain is not a valid domain name, it returns the ErrDomainNameNotValid
// error. If the property does not exist, or it does not belong to the current
// customer, it returns the ErrPropertyNotFound error.
func (this *Properties) RemoveDomain(id int, domain string) error {
	if id < 1 {
		panic("apis: invalid property identifier")
	}
	if !isValidDomainName(domain) {
		return ErrDomainNameNotValid
	}
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var customer int
		err := tx.QueryRow("SELECT `customer` FROM `properties` WHERE `id` = ?", id).Scan(&customer)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrPropertyNotFound
			}
			return err
		}
		if customer != this.customer {
			return ErrPropertyNotFound
		}
		_, err = tx.Table("Domains").Delete(sql.Where{"property": id, "domain": domain})
		return err
	})
	return err
}

// generatePropertyCode generates a property code.
func generatePropertyCode() (string, error) {
	var id uint64
	for id == 0 {
		i, err := rand.Int(rand.Reader, big.NewInt(3656158440062975))
		if err != nil {
			return "", err
		}
		id = i.Uint64()
	}
	var stringID = strings.ToUpper(strconv.FormatInt(int64(id), 36))
	if len(stringID) < 10 {
		stringID = fmt.Sprintf("%010s", stringID)
	}
	return stringID, nil
}

// isValidDomainName reports whether domain is a valid domain name.
func isValidDomainName(domain string) bool {

	var parts = strings.Split(domain, ".")
	if len(parts) != 2 {
		return false
	}

	var name = parts[0]
	var tld = parts[1]

	var minLen, maxLen int

	switch tld {
	case "it", "com":
		minLen = 3
		maxLen = 63
	default:
	}

	if len(name) < minLen || len(name) > maxLen || strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return false
	}

	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || c == '-':
		default:
			return false
		}
	}

	return true
}
