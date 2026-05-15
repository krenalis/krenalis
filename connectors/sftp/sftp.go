// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package sftp provides a connector for SFTP.
// (https://datatracker.ietf.org/doc/html/draft-ietf-secsh-filexfer-02)
package sftp

import (
	"bytes"
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

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

var errHostKeyMismatch = errors.New("host key mismatch")

func init() {
	connectors.RegisterFileStorage(connectors.FileStorageSpec{
		Code:       "sftp",
		Label:      "SFTP",
		Categories: connectors.CategoryFileStorage,
		AsSource: &connectors.AsFileStorageSource{
			Documentation: connectors.RoleDocumentation{
				Overview: sourceOverview,
			},
		},
		AsDestination: &connectors.AsFileStorageDestination{
			Documentation: connectors.RoleDocumentation{
				Overview: destinationOverview,
			},
		},
	}, New)
}

// New returns a new connector instance for SFTP.
func New(env *connectors.FileStorageEnv) (*SFTP, error) {
	return &SFTP{env: env}, nil
}

type SFTP struct {
	env *connectors.FileStorageEnv
}

type innerSettings struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	HostPublicKey string `json:"hostPublicKey"`
	TempPath      string `json:"tempPath"`
}

// AbsolutePath returns the absolute representation of the given path name.
func (sf *SFTP) AbsolutePath(ctx context.Context, name string) (string, error) {
	var s innerSettings
	err := sf.env.Settings.Load(ctx, &s)
	if err != nil {
		return "", err
	}
	u := url.URL{
		Scheme: "sftp",
		Host:   net.JoinHostPort(s.Host, strconv.Itoa(s.Port)),
		Path:   name,
	}
	return u.String(), nil
}

// Reader opens a file and returns a ReadCloser from which to read its content.
func (sf *SFTP) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	var s innerSettings
	err := sf.env.Settings.Load(ctx, &s)
	if err != nil {
		return nil, time.Time{}, err
	}
	client, err := openClient(ctx, &s)
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
func (sf *SFTP) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		err := sf.env.Settings.Load(ctx, &s)
		if err != nil {
			return nil, err
		}
		if s.Port == 0 {
			s.Port = 22
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, sf.saveSettings(ctx, settings, role, false)
	case "test":
		return nil, sf.saveSettings(ctx, settings, role, true)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{Name: "host", Label: "Host", Placeholder: "ftp.example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&connectors.Input{Name: "port", Label: "Port", Placeholder: "22", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5},
			&connectors.Input{Name: "username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 200},
			&connectors.Input{Name: "password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 200},
			&connectors.Input{Name: "hostPublicKey", Label: "Server public key", Placeholder: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...", HelpText: "Strongly recommended. Ensures the SFTP server identity matches this OpenSSH public key.", Rows: 3, MaxLength: 5000},
			&connectors.Input{Name: "tempPath", Label: "Temporary directory path", Placeholder: "/", Type: "text", MinLength: 0, MaxLength: 1000, Role: connectors.Destination},
		},
		Settings: settings,
		Buttons: []connectors.Button{
			{Event: "test", Text: "Test connection", Variant: "neutral"},
			connectors.SaveButton,
		},
	}

	return ui, nil
}

// Write writes the data read from r into the file with the given path name.
func (sf *SFTP) Write(ctx context.Context, r io.Reader, name, _ string) error {
	var s innerSettings
	err := sf.env.Settings.Load(ctx, &s)
	if err != nil {
		return err
	}
	client, err := openClient(ctx, &s)
	if err != nil {
		return err
	}
	defer client.close()
	if name[0] != '/' {
		name = "/" + name
	}
	if s.TempPath == "" {
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
	tempPath := s.TempPath
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
func (sf *SFTP) saveSettings(ctx context.Context, settings json.Value, role connectors.Role, test bool) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return connectors.NewInvalidSettingsError("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65535 {
		return connectors.NewInvalidSettingsError("port must be in range [1,65535]")
	}
	// Validate Username.
	if n := utf8.RuneCountInString(s.Username); n < 1 || n > 200 {
		return connectors.NewInvalidSettingsError("username length must be in range [1,200]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 200 {
		return connectors.NewInvalidSettingsError("password length must be in range [1,200]")
	}
	// Validate HostPublicKey.
	s.HostPublicKey = strings.TrimSpace(s.HostPublicKey)
	if s.HostPublicKey != "" {
		if n := utf8.RuneCountInString(s.HostPublicKey); n > 5000 {
			return connectors.NewInvalidSettingsError("server public key length must be in range [0,5000]")
		}
		if err := validateHostPublicKey(s.HostPublicKey); err != nil {
			return connectors.NewInvalidSettingsError(err.Error())
		}
	}
	// Validate TempPath.
	if role == connectors.Destination {
		if n := utf8.RuneCountInString(s.TempPath); n > 1000 {
			return connectors.NewInvalidSettingsError("length of temporary directory path must be in range [1,1000]")
		}
	} else if s.TempPath != "" {
		return connectors.NewInvalidSettingsError("temporary directory path must be empty for source destinations")
	}
	err = testConnection(ctx, &s)
	if err != nil || test {
		return err
	}
	return sf.env.Settings.Store(ctx, &s)
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

// hostKeyAlgorithms returns the host key algorithms accepted for the given key.
//
// RSA public keys are identified as "ssh-rsa", but SSH servers may use the
// same key with the stronger rsa-sha2-256 or rsa-sha2-512 signature algorithms
// during host key verification.
//
// For RSA keys, accept both RSA-SHA2 algorithms and the legacy ssh-rsa
// algorithm to remain compatible with older and newer servers.
func hostKeyAlgorithms(key ssh.PublicKey) []string {
	if key.Type() == ssh.KeyAlgoRSA {
		return []string{ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSA}
	}
	return []string{key.Type()}
}

// hostKeyValidator returns an SSH HostKeyCallback that verifies the server
// host key matches the expected key exactly. Connections using a different
// host key are rejected.
func hostKeyValidator(key ssh.PublicKey) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, candidate ssh.PublicKey) error {
		if key == nil {
			return errors.New("required host key was nil")
		}
		if !bytes.Equal(candidate.Marshal(), key.Marshal()) {
			return errHostKeyMismatch
		}
		return nil
	}
}

// newSSHClientConfig returns the SSH client configuration for s.
func newSSHClientConfig(s *innerSettings) (*ssh.ClientConfig, error) {
	sshConfig := &ssh.ClientConfig{
		User: s.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if s.HostPublicKey == "" {
		return sshConfig, nil
	}
	hostPublicKey, err := parseHostPublicKey(s.HostPublicKey)
	if err != nil {
		return nil, err
	}
	sshConfig.HostKeyCallback = hostKeyValidator(hostPublicKey)
	sshConfig.HostKeyAlgorithms = hostKeyAlgorithms(hostPublicKey)
	return sshConfig, nil
}

// openClient opens a client for the SFTP server based on the provided settings.
// The returned client must be closed using the close method. If the context is
// canceled before the client is closed, the underlying network connection, not
// the client, will be automatically closed.
func openClient(ctx context.Context, s *innerSettings) (*client, error) {
	sshConfig, err := newSSHClientConfig(s)
	if err != nil {
		return nil, err
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
		if errors.Is(err, errHostKeyMismatch) {
			return nil, connectors.NewInvalidSettingsError("server public key does not match the server host key")
		}
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

// parseHostPublicKey parses k as a single-line OpenSSH public key without
// authorized_keys options. The returned error messages are suitable for
// connectors.NewInvalidSettingsError.
func parseHostPublicKey(k string) (ssh.PublicKey, error) {
	if strings.ContainsAny(k, "\r\n") {
		return nil, errors.New("public key must be a single line")
	}
	key, _, options, rest, err := ssh.ParseAuthorizedKey([]byte(k))
	if err != nil {
		return nil, errors.New("server public key must be a valid OpenSSH public key")
	}
	if len(options) != 0 {
		return nil, errors.New("public key options are not allowed")
	}
	if len(rest) != 0 {
		return nil, errors.New("unexpected trailing data after public key")
	}
	return key, nil
}

// testConnection tests a connection using the provided settings. It returns an
// error if the connection cannot be established or if the server does not
// respond within 5 seconds.
func testConnection(ctx context.Context, settings *innerSettings) error {
	openCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	client, err := openClient(openCtx, settings)
	if err != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
		if openCtx.Err() != nil {
			return errors.New("sftp: unable to establish a connection within the 5-second limit. Verify the correctness of the settings and the functionality of the server")
		}
		return err
	}
	defer client.close()
	if settings.TempPath != "" {
		if _, ok := client.sftp.HasExtension("posix-rename@openssh.com"); !ok {
			return connectors.NewInvalidSettingsError("temporary directory path must be empty because the server does not support posix-rename")
		}
	}
	return nil
}

// validateHostPublicKey checks that k is a single-line OpenSSH public key
// without authorized_keys options.
func validateHostPublicKey(k string) error {
	_, err := parseHostPublicKey(k)
	return err
}
