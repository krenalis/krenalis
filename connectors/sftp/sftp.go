//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package sftp

// This package is the SFTP connector.
// (https://datatracker.ietf.org/doc/html/draft-ietf-secsh-filexfer-02)

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"time"
	"unicode/utf8"

	"chichi/connector"
	"chichi/connector/ui"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Connector icon.
var icon []byte

// Make sure it implements the StorageConnection interface.
var _ connector.StorageConnection = &connection{}

func init() {
	connector.RegisterStorage(connector.Storage{
		Name:    "SFTP",
		Icon:    icon,
		Connect: connect,
	})
}

type connection struct {
	ctx      context.Context
	settings *settings
	firehose connector.Firehose
}

type settings struct {
	Host     string
	Port     int
	Username string
	Password string
}

// connect returns a new SFTP connection.
func connect(ctx context.Context, conf *connector.StorageConfig) (connector.StorageConnection, error) {
	c := connection{ctx: ctx, firehose: conf.Firehose}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of SFTP connection")
		}
	}
	return &c, nil
}

// Reader returns a ReadCloser from which to read the file with the given path
// and its last update time.
// It is the caller's responsibility to close the returned reader.
func (c *connection) Reader(path string) (io.ReadCloser, time.Time, error) {
	if path == "" {
		return nil, time.Time{}, ui.Errorf("path is empty")
	}
	sshClient, sftpClient, err := openConnection(c.settings)
	if err != nil {
		return nil, time.Time{}, err
	}
	f, err := sftpClient.Open(path)
	if err != nil {
		_ = closeConnection(sshClient, sftpClient)
		return nil, time.Time{}, err
	}
	st, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}
	ts := st.ModTime().UTC()
	return reader{ssh: sshClient, sftp: sftpClient, fi: f}, ts, nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error) {

	var s settings

	switch event {
	case "load":
		// Load the Form.
		if c.settings == nil {
			s.Port = 22
		} else {
			s = *c.settings
		}
	case "test", "save":
		// Test the connection and save the settings if required.
		err := json.Unmarshal(values, &s)
		if err != nil {
			return nil, nil, err
		}
		// Validate Host.
		if n := len(s.Host); n == 0 || n > 253 {
			return nil, nil, ui.Errorf("host length in bytes must be in range [1,253]")
		}
		// Validate Port.
		if s.Port < 1 || s.Port > 65536 {
			return nil, nil, ui.Errorf("port must be in range [1,65536]")
		}
		// Validate Username.
		if n := utf8.RuneCountInString(s.Username); n < 1 || n > 200 {
			return nil, nil, ui.Errorf("username length must be in range [1,200]")
		}
		// Validate Password.
		if n := utf8.RuneCountInString(s.Password); n < 1 || n > 200 {
			return nil, nil, ui.Errorf("password length must be in range [1,200]")
		}
		err = testConnection(&s)
		if err != nil {
			if event == "test" {
				return nil, ui.WarningAlert(err.Error()), nil
			}
			return nil, ui.DangerAlert(err.Error()), nil
		}
		if event == "test" {
			return nil, ui.SuccessAlert("Connection established"), nil
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, nil, err
		}
		err = c.firehose.SetSettings(b)
		if err != nil {
			return nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "host", Value: s.Host, Label: "Host", Placeholder: "ftp.example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&ui.Input{Name: "port", Value: s.Port, Label: "Port", Placeholder: "22", Type: "number", MinLength: 1, MaxLength: 5},
			&ui.Input{Name: "username", Value: s.Username, Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 200},
			&ui.Input{Name: "password", Value: s.Password, Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 200},
		},
		Actions: []ui.Action{
			{Event: "test", Text: "Test Connection", Variant: "neutral"},
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// Write writes the data read from p into the file with the given path.
func (c *connection) Write(r io.Reader, path, _ string) error {
	if path == "" {
		return ui.Errorf("path is empty")
	}
	sshClient, sftpClient, err := openConnection(c.settings)
	if err != nil {
		return err
	}
	f, err := sftpClient.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		_ = closeConnection(sshClient, sftpClient)
		return err
	}
	_, err = io.Copy(f, r)
	err2 := f.Close()
	err3 := closeConnection(sshClient, sftpClient)
	if err != nil {
		return err
	}
	if err2 != nil {
		return err2
	}
	return err3
}

type reader struct {
	ssh  *ssh.Client
	sftp *sftp.Client
	fi   *sftp.File
}

func (r reader) Close() error {
	err := r.fi.Close()
	err2 := closeConnection(r.ssh, r.sftp)
	r.ssh = nil
	r.sftp = nil
	if err != nil {
		return err
	}
	return err2
}

func (r reader) Read(p []byte) (int, error) {
	return r.fi.Read(p)
}

// openConnection opens the connection.
func openConnection(s *settings) (*ssh.Client, *sftp.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User: s.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO(marco) don't use in production
	}
	addr := s.Host + ":" + strconv.Itoa(s.Port)
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, nil, err
	}
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, nil, err
	}
	return sshClient, sftpClient, nil
}

// closeConnection closes the connection.
func closeConnection(sshClient *ssh.Client, sftpClient *sftp.Client) error {
	err := sftpClient.Close()
	err2 := sshClient.Close()
	if err != nil {
		return err
	}
	return err2
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func testConnection(settings *settings) error {
	sshClient, sftpClient, err := openConnection(settings)
	if err != nil {
		return err
	}
	_ = closeConnection(sshClient, sftpClient)
	return nil
}
