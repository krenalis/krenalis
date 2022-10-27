//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package sftp

// This package is the SFTP connector.
// (https://datatracker.ietf.org/doc/html/draft-ietf-secsh-filexfer-02)

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"time"

	"chichi/connectors"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Make sure it implements the StreamConnection interface.
var _ connectors.StreamConnection = &connection{}

func init() {
	connectors.RegisterStreamConnector("SFTP", New)
}

type connection struct {
	ctx      context.Context
	settings *settings
	ssh      *ssh.Client
	sftp     *sftp.Client
}

type settings struct {
	Host     string
	Port     int
	Username string
	Password string
	Path     string
}

// New returns a new SFTP connection.
func New(ctx context.Context, settings []byte, fh connectors.Firehose) (connectors.StreamConnection, error) {
	c := connection{ctx: ctx}
	if len(settings) > 0 {
		err := json.Unmarshal(settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of SFTP connection")
		}
	}
	return &c, nil
}

// Reader returns a ReadCloser from which to read the data and its last update
// time.
// It is the caller's responsibility to close the returned reader.
func (c *connection) Reader() (io.ReadCloser, time.Time, error) {
	err := c.openConnection()
	if err != nil {
		return nil, time.Time{}, err
	}
	f, err := c.sftp.Open(c.settings.Path)
	if err != nil {
		_ = c.closeConnection()
		return nil, time.Time{}, err
	}
	st, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}
	ts := st.ModTime().UTC()
	return reader{c, f}, ts, nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, form []byte) (*connectors.SettingsUI, error) {
	return nil, nil
}

// Write writes the data read from p.
func (c *connection) Write(r io.Reader) error {
	err := c.openConnection()
	if err != nil {
		return err
	}
	f, err := c.sftp.OpenFile(c.settings.Path, os.O_RDONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		_ = c.closeConnection()
		return err
	}
	_, err = io.Copy(f, r)
	err2 := f.Close()
	err3 := c.closeConnection()
	if err != nil {
		return err
	}
	if err2 != nil {
		return err2
	}
	return err3
}

type reader struct {
	c  *connection
	fi *sftp.File
}

func (r reader) Close() error {
	err := r.fi.Close()
	err2 := r.c.closeConnection()
	if err != nil {
		return err
	}
	return err2
}

func (r reader) Read(p []byte) (int, error) {
	return r.fi.Read(p)
}

// openConnection opens the connection.
func (c *connection) openConnection() error {
	sshConfig := &ssh.ClientConfig{
		User: c.settings.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.settings.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO(marco) don't use in production
	}
	addr := c.settings.Host + ":" + strconv.Itoa(c.settings.Port)
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return err
	}
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return err
	}
	c.ssh = sshClient
	c.sftp = sftpClient
	return nil
}

// closeConnection closes the connection.
func (c *connection) closeConnection() error {
	if c.sftp == nil {
		return nil
	}
	err := c.sftp.Close()
	c.sftp = nil
	err2 := c.ssh.Close()
	c.ssh = nil
	if err != nil {
		return err
	}
	return err2
}
