//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package sftp implements the SFTP connector.
// (https://datatracker.ietf.org/doc/html/draft-ietf-secsh-filexfer-02)
package sftp

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi"
	"chichi/ui"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the UI and the StorageConnection interfaces.
var _ interface {
	chichi.UI
	chichi.StorageConnection
} = (*connection)(nil)

func init() {
	chichi.RegisterStorage(chichi.Storage{
		Name: "SFTP",
		Icon: icon,
	}, new)
}

// new returns a new SFTP connection.
func new(conf *chichi.StorageConfig) (*connection, error) {
	c := connection{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of SFTP connection")
		}
	}
	return &c, nil
}

type connection struct {
	conf     *chichi.StorageConfig
	settings *settings
}

type settings struct {
	Host     string
	Port     int
	Username string
	Password string
	TempPath string
}

// CompletePath returns the complete representation of the given path name or an
// InvalidPathError if name is not valid for use in calls to Reader and Write.
func (c *connection) CompletePath(ctx context.Context, name string) (string, error) {
	u := url.URL{
		Scheme: "sftp",
		Host:   net.JoinHostPort(c.settings.Host, strconv.Itoa(c.settings.Port)),
		Path:   name,
	}
	return u.String(), nil
}

// Reader opens the file at the given path name and returns a ReadCloser from
// which to read the file and its last update time. The use of the provided
// context is extended to the Read method calls. After the context is canceled,
// any subsequent Read invocations will result in an error.
// It is the caller's responsibility to close the returned reader.
func (c *connection) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	client, err := openClient(ctx, c.settings)
	if err != nil {
		return nil, time.Time{}, err
	}
	var r io.ReadCloser
	defer func() {
		// Close the client if there was an error or a panic.
		if r == nil {
			_ = client.close()
		}
	}()
	if name[0] != '/' {
		name = "/" + name
	}
	f, err := client.sftp.Open(name)
	if err != nil {
		return nil, time.Time{}, err
	}
	st, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}
	ts := st.ModTime().UTC()
	r = reader{client: client, fi: f}
	return r, ts, nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(ctx context.Context, event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the Form.
		var s settings
		if c.settings == nil {
			s.Port = 22
		} else {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "test", "save":
		// Test the connection and save the settings if required.
		s, err := c.ValidateSettings(ctx, values)
		if err != nil {
			if event == "test" {
				return nil, ui.WarningAlert(err.Error()), nil
			}
			return nil, ui.DangerAlert(err.Error()), nil
		}
		if event == "test" {
			return nil, ui.SuccessAlert("Connection established"), nil
		}
		err = c.conf.SetSettings(ctx, s)
		if err != nil {
			return nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "host", Label: "Host", Placeholder: "ftp.example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&ui.Input{Name: "port", Label: "Port", Placeholder: "22", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5},
			&ui.Input{Name: "username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 200},
			&ui.Input{Name: "password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 200},
			&ui.Input{Name: "tempPath", Label: "Temporary directory path", Placeholder: "/", Type: "text", MinLength: 0, MaxLength: 1000, Role: ui.Destination},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "test", Text: "Test Connection", Variant: "neutral"},
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (c *connection) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return nil, ui.Errorf("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65536 {
		return nil, ui.Errorf("port must be in range [1,65536]")
	}
	// Validate Username.
	if n := utf8.RuneCountInString(s.Username); n < 1 || n > 200 {
		return nil, ui.Errorf("username length must be in range [1,200]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 200 {
		return nil, ui.Errorf("password length must be in range [1,200]")
	}
	// Validate TempPath.
	if c.conf.Role == chichi.Destination {
		if n := utf8.RuneCountInString(s.TempPath); n > 1000 {
			return nil, ui.Errorf("length of temporary directory path must be in range [1,1000]")
		}
	} else if s.TempPath != "" {
		return nil, ui.Errorf("temporary directory path must be empty for source destinations")
	}
	err = testConnection(ctx, &s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&s)
}

// Write writes the data read from r into the file with the given path name.
// contentType is the file's content type.
func (c *connection) Write(ctx context.Context, r io.Reader, name, _ string) error {
	client, err := openClient(ctx, c.settings)
	if err != nil {
		return err
	}
	defer client.close()
	if name[0] != '/' {
		name = "/" + name
	}
	if c.settings.TempPath == "" {
		var f *sftp.File
		f, err = client.sftp.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
		if err != nil {
			return err
		}
		if _, err = io.Copy(f, r); err != nil {
			_ = f.Close()
			return err
		}
		if err = f.Close(); err != nil {
			return err
		}
		return client.close()
	}
	// Create the file atomically.
	base := path.Base(name)
	ext := path.Ext(name)
	tempPath := c.settings.TempPath
	if tempPath[0] != '/' {
		tempPath = "/" + tempPath
	}
	tempName := path.Join(tempPath, strings.TrimSuffix(base, ext)) + "-" + strconv.FormatUint(rand.Uint64(), 10) + ext
	f, err := client.sftp.OpenFile(tempName, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = client.sftp.Remove(tempName)
		}
	}()
	if _, err = io.Copy(f, r); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = client.sftp.PosixRename(tempName, name); err != nil {
		return err
	}
	return client.close()
}

type reader struct {
	client *client
	fi     *sftp.File
}

func (r reader) Close() error {
	if r.client == nil {
		return nil
	}
	defer r.client.close()
	err := r.fi.Close()
	r.fi = nil
	err2 := r.client.close()
	r.client = nil
	if err != nil {
		return err
	}
	return err2
}

func (r reader) Read(p []byte) (int, error) {
	return r.fi.Read(p)
}

// client represents an SFTP client.
type client struct {
	ssh  *ssh.Client
	sftp *sftp.Client

	// stop stops the association of the context with the function that closes
	// the underlying connection. It is nil if the client is closed.
	stop func() bool
}

// close closes the client.
// It does nothing if the client has already been closed.
func (client *client) close() error {
	if client.stop == nil {
		return nil
	}
	defer func() {
		client.stop()
		client.stop = nil
	}()
	err := client.sftp.Close()
	err2 := client.ssh.Close()
	if err != nil {
		return err
	}
	return err2
}

// openClient opens a client for the SFTP server based on the provided settings.
// The returned client must be closed using the close method. If the context is
// canceled before the client is closed, the underlying network connection, not
// the client, will be automatically closed.
func openClient(ctx context.Context, s *settings) (*client, error) {
	sshConfig := &ssh.ClientConfig{
		User: s.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO(marco) don't use in production
	}
	addr := s.Host + ":" + strconv.Itoa(s.Port)
	// The ssh package does not implement the ssh.DialContext function
	// (see issue https://github.com/golang/go/issues/64686), and the
	// ssh.NewClientConn function does not accept a context, so we have to close
	// the connection passed to it to stop its execution if the context is
	// canceled.
	d := &net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	stop := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})
	// After this point, if the context is canceled, the underlying connection
	// will be closed.
	var cl *client
	defer func() {
		// Close the connection if there was an error or a panic.
		if cl == nil {
			stop()
		}
	}()
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, sshConfig)
	if err != nil {
		return nil, err
	}
	sshClient := ssh.NewClient(c, chans, reqs)
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, err
	}
	cl = &client{ssh: sshClient, sftp: sftpClient, stop: stop}
	return cl, nil
}

// testConnection tests a connection using the provided settings. It returns an
// error if the connection cannot be established or if the server does not
// respond within 5 seconds.
func testConnection(ctx context.Context, settings *settings) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	client, err := openClient(ctx, settings)
	if err != nil {
		if err := ctx.Err(); err != nil {
			return errors.New("sftp: unable to establish a connection within the 5-second limit. Verify the correctness of the settings and the functionality of the server")
		}
		return err
	}
	defer client.close()
	if settings.TempPath != "" {
		if _, ok := client.sftp.HasExtension("posix-rename@openssh.com"); !ok {
			return ui.Errorf("temporary directory path must be empty because the server does not support posix-rename")
		}
	}
	return nil
}
