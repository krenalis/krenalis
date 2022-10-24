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
	"net/http"
	"os"
	"strconv"

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
	User     string
	Password string
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

// Reader returns a Reader that read from the given path.
func (c *connection) Reader(path string) (io.ReadCloser, error) {
	err := c.openConnection()
	if err != nil {
		return nil, err
	}
	f, err := c.sftp.Open(path)
	if err != nil {
		_ = c.closeConnection()
		return nil, err
	}
	return reader{c, f}, nil
}

// ServeUserInterface serves the connector's user interface.
func (c *connection) ServeUserInterface(w http.ResponseWriter, r *http.Request) {}

// Writer returns a Writer that writes to the given path.
func (c *connection) Writer(path string) (io.WriteCloser, error) {
	err := c.openConnection()
	if err != nil {
		return nil, err
	}
	f, err := c.sftp.OpenFile(path, os.O_RDONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		_ = c.closeConnection()
		return nil, err
	}
	return writer{c, f}, nil
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

type writer struct {
	c  *connection
	fi *sftp.File
}

func (w writer) Close() error {
	err := w.fi.Close()
	err2 := w.c.closeConnection()
	if err != nil {
		return err
	}
	return err2
}

func (w writer) Write(p []byte) (int, error) {
	return w.fi.Write(p)
}

// openConnection opens the connection.
func (c *connection) openConnection() error {
	sshConfig := &ssh.ClientConfig{
		User: c.settings.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.settings.Password),
		},
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
