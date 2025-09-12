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
	"errors"
	"io"
	"math/rand/v2"
	"net"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/json"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Connector icon.
var icon = "<svg></svg>"

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	meergo.RegisterFileStorage(meergo.FileStorageInfo{
		Name:       "SFTP",
		Categories: meergo.CategoryFileStorage,
		AsSource: &meergo.AsFileStorageSource{
			Documentation: meergo.ConnectorRoleDocumentation{
				Overview: sourceOverview,
			},
		},
		AsDestination: &meergo.AsFileStorageDestination{
			Documentation: meergo.ConnectorRoleDocumentation{
				Overview: destinationOverview,
			},
		},
		Icon: icon,
	}, New)
}

// New returns a new SFTP connector instance.
func New(env *meergo.FileStorageEnv) (*SFTP, error) {
	c := SFTP{env: env}
	if len(env.Settings) > 0 {
		err := json.Value(env.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of SFTP connector")
		}
	}
	return &c, nil
}

type SFTP struct {
	env      *meergo.FileStorageEnv
	settings *innerSettings
}

type innerSettings struct {
	Host     string
	Port     int
	Username string
	Password string
	TempPath string
}

// AbsolutePath returns the absolute representation of the given path name.
func (sf *SFTP) AbsolutePath(ctx context.Context, name string) (string, error) {
	u := url.URL{
		Scheme: "sftp",
		Host:   net.JoinHostPort(sf.settings.Host, strconv.Itoa(sf.settings.Port)),
		Path:   name,
	}
	return u.String(), nil
}

// Reader opens a file and returns a ReadCloser from which to read its content.
func (sf *SFTP) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	client, err := openClient(ctx, sf.settings)
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
		_ = f.Close()
		return nil, time.Time{}, err
	}
	ts := st.ModTime().UTC()
	r = reader{client: client, fi: f}
	return r, ts, nil
}

// ServeUI serves the connector's user interface.
func (sf *SFTP) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if sf.settings == nil {
			s.Port = 22
		} else {
			s = *sf.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, sf.saveSettings(ctx, settings, role, false)
	case "test":
		return nil, sf.saveSettings(ctx, settings, role, true)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "Host", Label: "Host", Placeholder: "ftp.example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&meergo.Input{Name: "Port", Label: "Port", Placeholder: "22", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5},
			&meergo.Input{Name: "Username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 200},
			&meergo.Input{Name: "Password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 200},
			&meergo.Input{Name: "TempPath", Label: "Temporary directory path", Placeholder: "/", Type: "text", MinLength: 0, MaxLength: 1000, Role: meergo.Destination},
		},
		Settings: settings,
		Buttons: []meergo.Button{
			{Event: "test", Text: "Test connection", Variant: "neutral"},
		},
	}

	return ui, nil
}

// Write writes the data read from r into the file with the given path name.
func (sf *SFTP) Write(ctx context.Context, r io.Reader, name, _ string) error {
	client, err := openClient(ctx, sf.settings)
	if err != nil {
		return err
	}
	defer client.close()
	if name[0] != '/' {
		name = "/" + name
	}
	if sf.settings.TempPath == "" {
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
	tempPath := sf.settings.TempPath
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

// saveSettings validates and saves the settings. If test is true, it validates
// only the settings without saving it.
func (sf *SFTP) saveSettings(ctx context.Context, settings json.Value, role meergo.Role, test bool) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return meergo.NewInvalidSettingsError("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65535 {
		return meergo.NewInvalidSettingsError("port must be in range [1,65535]")
	}
	// Validate Username.
	if n := utf8.RuneCountInString(s.Username); n < 1 || n > 200 {
		return meergo.NewInvalidSettingsError("username length must be in range [1,200]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 200 {
		return meergo.NewInvalidSettingsError("password length must be in range [1,200]")
	}
	// Validate TempPath.
	if role == meergo.Destination {
		if n := utf8.RuneCountInString(s.TempPath); n > 1000 {
			return meergo.NewInvalidSettingsError("length of temporary directory path must be in range [1,1000]")
		}
	} else if s.TempPath != "" {
		return meergo.NewInvalidSettingsError("temporary directory path must be empty for source destinations")
	}
	err = testConnection(ctx, &s)
	if err != nil || test {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = sf.env.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	sf.settings = &s
	return nil
}

type reader struct {
	client *client
	fi     *sftp.File
}

func (r reader) Close() error {
	defer r.client.close()
	err := r.fi.Close()
	if err != nil {
		return err
	}
	return r.client.close()
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
func openClient(ctx context.Context, s *innerSettings) (*client, error) {
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
func testConnection(ctx context.Context, settings *innerSettings) error {
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
			return meergo.NewInvalidSettingsError("temporary directory path must be empty because the server does not support posix-rename")
		}
	}
	return nil
}
