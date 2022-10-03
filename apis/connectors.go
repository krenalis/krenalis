package apis

import (
	"chichi/pkg/open2b/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Connectors struct {
	*APIs
}

type Connector struct {
	ID            int
	Name          string
	OauthURL      string
	LogoURL       string
	ClientID      string
	ClientSecret  string
	TokenEndpoint string
}

func (this *Connectors) Find() ([]*Connector, error) {
	connectors := make([]*Connector, 0, 0)
	err := this.myDB.QueryScan("SELECT `id`, `name`, `oauth_url`, `logo_url`\nFROM `connectors`", func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var connector Connector
			if err = rows.Scan(&connector.ID, &connector.Name, &connector.OauthURL, &connector.LogoURL); err != nil {
				return err
			}
			connectors = append(connectors, &connector)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connectors, nil
}

func (this *Connectors) Get(id int) (*Connector, error) {
	connector := Connector{ID: id}
	err := this.myDB.QueryRow("SELECT `name`, `oauth_url`, `logo_url`, `client_id`, `client_secret`, `token_endpoint`\nFROM `connectors`\nWHERE `id` = ?", id).
		Scan(&connector.Name, &connector.OauthURL, &connector.LogoURL, &connector.ClientID, &connector.ClientSecret, &connector.TokenEndpoint)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &connector, nil
}

func (this *Connectors) FindAccountConnectors(accountID int) ([]*Connector, error) {
	ids := make([]int, 0, 0)
	err := this.myDB.QueryScan("SELECT `connector`\nFROM `account_connectors`\nWHERE account = ?", accountID, func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var id int
			if err = rows.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	connectors := make([]*Connector, 0, 0)
	if len(ids) == 0 {
		return connectors, nil
	}

	stringifiedIDs := make([]string, 0, 0)
	for _, id := range ids {
		stringifiedIDs = append(stringifiedIDs, strconv.Itoa(id))
	}

	err = this.myDB.QueryScan(fmt.Sprintf("SELECT `id`, `name`, `oauth_url`, `logo_url`\nFROM `connectors` WHERE id IN (%s)", strings.Join(stringifiedIDs, ", ")), func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var connector Connector
			if err = rows.Scan(&connector.ID, &connector.Name, &connector.OauthURL, &connector.LogoURL); err != nil {
				return err
			}
			connectors = append(connectors, &connector)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connectors, nil
}

func (this *Connectors) GetAccountConnector(accountID int, connectorID int) (string, string, *time.Time, error) {
	var accessToken, refreshToken string
	var expiration time.Time
	err := this.myDB.QueryRow("SELECT `access_token`, `refresh_token`, `access_token_expiration_timestamp`\nFROM `account_connectors`\nWHERE `account` = ? AND `connector` = ?", accountID, connectorID).
		Scan(&accessToken, &refreshToken, &expiration)
	if err != nil {
		return "", "", nil, err
	}
	return accessToken, refreshToken, &expiration, nil
}

func (this *Connectors) SaveAccountConnector(accountID int, connectorID int, accessToken string, refreshToken string, expiration time.Time) error {
	_, err := this.myDB.Exec("INSERT INTO `account_connectors` SET `account` = ?, `connector` = ?, `access_token` = ?, `refresh_token` = ?, `access_token_expiration_timestamp` = ?\nON DUPLICATE KEY UPDATE `access_token` = ?, `refresh_token` = ?, `access_token_expiration_timestamp` = ?",
		accountID, connectorID, accessToken, refreshToken, expiration, accessToken, refreshToken, expiration)
	if err != nil {
		return err
	}
	return nil
}

func (this *Connectors) DeleteAccountConnector(accountID int, ids []int) error {
	_, err := this.myDB.Table("AccountConnectors").Delete(sql.Where{"account": accountID, "connector": ids})
	if err != nil {
		return err
	}
	return nil
}
