//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	o2bsql "chichi/pkg/open2b/sql"
)

type SmartEvents struct {
	*Properties
}

type (
	SmartEventToCreate struct {
		Name    string
		Event   string
		Pages   []Condition // for both "view" and "click"
		Buttons []Condition // only for "click"
	}
	SmartEventToUpdate = SmartEventToCreate
	SmartEvent         struct {
		ID      int
		Name    string
		Event   string
		Pages   []Condition // for both "view" and "click"
		Buttons []Condition // only for "click"
	}
)

type InvalidSmartEventError string

func (err InvalidSmartEventError) Error() string {
	return fmt.Sprintf("invalid Smart Event: %s", string(err))
}

type DomainNotAllowedError string

func (err DomainNotAllowedError) Error() string {
	return fmt.Sprintf("domain %q is not allowed", string(err))
}

// Create creates a new Smart Event.
func (smartEvents *SmartEvents) Create(smartEvent SmartEventToCreate) (int64, error) {

	// Do some validations.
	switch smartEvent.Event {
	case "view":
		if smartEvent.Buttons != nil {
			return 0, InvalidSmartEventError("apis: Buttons must be 'null' when 'Event' is 'view")
		}
	case "click":
	default:
		panic(fmt.Sprintf("apis: unsupported event type %q", smartEvent.Event))
	}
	err := validateConditions(smartEvent.Pages)
	if err != nil {
		return 0, err
	}

	// If one of the conditions has a domain, then every other condition must
	// have the domain.
	var haveDomains bool
	if len(smartEvent.Pages) > 0 {
		haveDomains = smartEvent.Pages[0].Domain != ""
		for _, page := range smartEvent.Pages[1:] {
			if haveDomains != (page.Domain != "") {
				return 0, InvalidSmartEventError("cannot have both conditions with domain and no domains")
			}
		}
	}
	if len(smartEvent.Buttons) > 0 {
		for _, button := range smartEvent.Buttons[1:] {
			if haveDomains != (button.Domain != "") {
				return 0, InvalidSmartEventError("cannot have both conditions with domain and no domains")
			}
		}
	}

	// Collect the domains used in the conditions.
	var domains map[string]bool
	if haveDomains {
		domains = map[string]bool{}
	}
	for _, cond := range smartEvent.Pages {
		if cond.Domain != "" {
			domains[cond.Domain] = true
		}
	}
	for _, cond := range smartEvent.Buttons {
		if cond.Domain != "" {
			domains[cond.Domain] = true
		}
	}

	// Serialize the Smart Event to create and write it to the database.
	name, event, rawPages, rawButtons, err := serializeSmartEvent(smartEvent)
	if err != nil {
		return 0, err
	}

	var id int64
	err = smartEvents.myDB.Transaction(func(tx *o2bsql.Tx) error {
		// Retrieve the list of rows for the current property.
		rows, err := tx.Table("Domains").Select(
			o2bsql.Columns{"name"},
			o2bsql.Where{"property": smartEvents.Properties.id},
			nil, 0, 0,
		).Rows()
		if err != nil {
			return fmt.Errorf("cannot retrieve the list of domains: %s", err)
		}
		allowedDomains := map[string]bool{}
		for _, row := range rows {
			allowedDomains[row["name"].(string)] = true
		}
		for domain := range domains {
			if !allowedDomains[domain] {
				return DomainNotAllowedError(domain)
			}
		}

		query := "INSERT INTO `smart_events` (`property`, `name`, `event`, `pages`, `buttons`) VALUES (?, ?, ?, ?, ?)"
		result, err := smartEvents.myDB.Exec(query, smartEvents.Properties.id, name, event, rawPages, rawButtons)
		if err != nil {
			return err
		}
		id, err = result.LastInsertId()
		return err
	})
	if err != nil {
		return 0, err
	}

	return id, nil
}

// Delete deletes the Smart Events with the given IDs.
func (smartEvents *SmartEvents) Delete(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	in := &strings.Builder{}
	in.WriteString("(")
	for i, id := range ids {
		if id <= 0 {
			panic("apis: IDs must be > 0")
		}
		if i > 0 {
			in.WriteString(", ")
		}
		in.WriteString(strconv.Itoa(id))
	}
	in.WriteString(")")
	query := "DELETE FROM `smart_events` WHERE `id` IN " + in.String() + "AND `property` = ?"
	_, err := smartEvents.myDB.Exec(query, smartEvents.Properties.id)
	return err
}

// Find finds the Smart Events.
func (smartEvents *SmartEvents) Find() ([]SmartEvent, error) {
	query := "SELECT `id`, `name`, `event`, `pages`, `buttons` FROM `smart_events` WHERE `property` = ? ORDER BY `id`"
	rows, err := smartEvents.myDB.Query(query, smartEvents.Properties.id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []SmartEvent
	for rows.Next() {
		var id int
		var name, event string
		var rawPages, rawButtons string
		err := rows.Scan(&id, &name, &event, &rawPages, &rawButtons)
		if err != nil {
			return nil, err
		}
		smartEvent, err := deserializeSmartEvent(id, name, event, rawPages, rawButtons)
		if err != nil {
			return nil, err
		}
		events = append(events, smartEvent)
	}
	if rows.Err() != nil {
		return nil, err
	}
	return events, nil
}

// Get gets the Smart Event with the given ID. If the ID does not correspond to
// any Smart Event, this method returns SmartEvent{}, nil.
func (smartEvents *SmartEvents) Get(id int) (SmartEvent, error) {
	if id <= 0 {
		panic("apis: id must be > 0")
	}
	query := "SELECT `name`, `event`, `pages`, `buttons` FROM `smart_events` WHERE `id` = ? AND `property` = ?"
	row := smartEvents.myDB.QueryRow(query, id, smartEvents.Properties.id)
	var name, event string
	var rawPages, rawButtons string
	err := row.Scan(&name, &event, &rawPages, &rawButtons)
	if err != nil {
		if err == sql.ErrNoRows {
			return SmartEvent{}, nil
		}
		return SmartEvent{}, err
	}
	return deserializeSmartEvent(id, name, event, rawPages, rawButtons)
}

func (smartEvents *SmartEvents) Update(id int, event SmartEventToUpdate) error {
	if id <= 0 {
		panic("apis: id must be > 0")
	}
	_, _, rawPages, rawButtons, err := serializeSmartEvent(event)
	if err != nil {
		return err
	}
	toUpdate := map[string]any{
		"name":    event.Name,
		"event":   event.Event,
		"pages":   rawPages,
		"buttons": rawButtons,
	}
	_, err = smartEvents.myDB.Table("SmartEvents").Update(toUpdate, o2bsql.Where{
		"id":       id,
		"property": smartEvents.Properties.id,
	})
	if err != nil {
		return err
	}
	return nil
}

// deserializeSmartEvent deserializes the components of a Smart Event and return
// a value of type SmartEvent.
func deserializeSmartEvent(id int, name, event string, rawPages, rawButtons string) (SmartEvent, error) {
	var pages, buttons []Condition
	err := json.Unmarshal([]byte(rawPages), &pages)
	if err != nil {
		return SmartEvent{}, err
	}
	err = json.Unmarshal([]byte(rawButtons), &buttons)
	if err != nil {
		return SmartEvent{}, err
	}
	smartEvent := SmartEvent{
		ID:      id,
		Name:    name,
		Event:   event,
		Pages:   pages,
		Buttons: buttons,
	}
	return smartEvent, nil
}

// serializeSmartEvent serializes the given Smart Event into its components,
// which can be written to the database.
func serializeSmartEvent(smartEvent SmartEventToCreate) (name, event string, rawPages, rawButtons string, err error) {
	rawPagesBytes, err := json.Marshal(smartEvent.Pages)
	if err != nil {
		return
	}
	rawButtonsBytes, err := json.Marshal(smartEvent.Buttons)
	if err != nil {
		return
	}
	return smartEvent.Name, smartEvent.Event, string(rawPagesBytes), string(rawButtonsBytes), nil
}

// validateCondition validate the given conditions, returning error if one of
// them it's not valid.
// Returns nil or an InvalidSmartEventError.
func validateConditions(conditions []Condition) error {
	// TODO(Gianluca): add more validations here.
	for _, cond := range conditions {
		if cond.Field == "" {
			return InvalidSmartEventError("field 'Field' cannot be empty")
		}
		if cond.Operator == "" {
			return InvalidSmartEventError("field 'Operator' cannot be empty")
		}
		if cond.Value == "" {
			return InvalidSmartEventError("field 'Value' cannot be empty")
		}
	}
	return nil
}
