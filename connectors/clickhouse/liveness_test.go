// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package clickhouse

import (
	"context"
	"io"
	"net"
	"strconv"
	"sync"
	"syscall"
	"testing"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/json"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/clickhouse"
)

// Test_PooledConnectionLiveness checks that the ClickHouse driver still detects
// a pooled connection dropped while idle when the dial function returns a
// wrapped net.Conn, as it does with counting enabled. Before reusing a pooled
// connection the driver asserts it to syscall.Conn and, if the assertion fails,
// declares it healthy without checking it (see conn_check.go). The wrapper must
// therefore keep the syscall.Conn of the connection it wraps, or the driver
// hands a dead connection to the next query.
func Test_PooledConnectionLiveness(t *testing.T) {
	const username, password, database = "test_krenalis", "test_krenalis", "test_krenalis"
	ctx := context.Background()
	container, err := clickhouse.Run(ctx,
		testimages.ClickHouse,
		clickhouse.WithUsername(username),
		clickhouse.WithPassword(password),
		clickhouse.WithDatabase(database),
	)
	defer func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("cannot terminate the container: %s", err)
		}
	}()
	if err != nil {
		t.Fatal(err)
	}
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "9000/tcp")
	if err != nil {
		t.Fatal(err)
	}
	target := net.JoinHostPort(host, port.Port())

	for _, tc := range []struct {
		name        string
		keepSyscall bool // whether the wrapper exposes the syscall.Conn of the connection
		wantErr     bool // whether the query on the dropped pooled connection fails
	}{
		{"wrapper keeps syscall.Conn", true, false},
		{"wrapper hides syscall.Conn", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The driver reaches ClickHouse through a proxy, so the test can drop
			// the connection the driver keeps in its idle pool without stopping
			// the server.
			proxy := newTCPProxy(t, target)
			proxyHost, proxyPort, err := net.SplitHostPort(proxy.addr)
			if err != nil {
				t.Fatal(err)
			}
			settings, err := json.Marshal(innerSettings{
				Host:     proxyHost,
				Port:     mustAtoi(t, proxyPort),
				Username: username,
				Password: password,
				Database: database,
			})
			if err != nil {
				t.Fatal(err)
			}

			dial := func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				conn, err := d.DialContext(ctx, network, addr)
				if err != nil {
					return nil, err
				}
				w := wrappedConn{conn}
				if !tc.keepSyscall {
					return w, nil
				}
				sc, ok := conn.(syscall.Conn)
				if !ok {
					t.Errorf("the dialed connection is a %T, which is not a syscall.Conn", conn)
					return w, nil
				}
				return wrappedSyscallConn{w, sc}, nil
			}
			env := connectors.DatabaseEnv{Settings: newTestSettingsStore(settings), Dial: dial}
			connector, err := New(&env)
			if err != nil {
				t.Fatal(err)
			}
			defer connector.Close()
			if err := connector.openDB(ctx); err != nil {
				t.Fatalf("cannot open the database: %s", err)
			}

			// The first query opens the pooled connection and returns it to the
			// pool.
			if err := connector.db.Exec(ctx, "SELECT 1"); err != nil {
				t.Fatalf("first query: %s", err)
			}
			dials := proxy.dials()

			// The pooled connection is dropped while idle.
			proxy.dropAll()

			err = connector.db.Exec(ctx, "SELECT 1")
			if tc.wantErr {
				if err == nil {
					t.Fatal("the query on the dropped pooled connection succeeded, expecting it to fail")
				}
				return
			}
			if err != nil {
				t.Fatalf("query after the idle connection was dropped: %s", err)
			}
			// The driver found the pooled connection dead, discarded it and
			// dialed a new one through the proxy.
			if n := proxy.dials(); n <= dials {
				t.Fatalf("the driver reused the dropped connection: %d dials before, %d after", dials, n)
			}
		})
	}
}

// wrappedConn is the shape of the connection wrapper that core/internal/dialer
// puts around a counted connection: it embeds net.Conn only.
type wrappedConn struct {
	net.Conn
}

// wrappedSyscallConn is wrappedConn for a connection that also implements
// syscall.Conn, which the dialer keeps exposed so the ClickHouse driver can
// check whether a pooled connection is still alive before reusing it.
type wrappedSyscallConn struct {
	wrappedConn
	syscallConn syscall.Conn
}

// SyscallConn returns the raw network connection of the wrapped connection.
func (c wrappedSyscallConn) SyscallConn() (syscall.RawConn, error) {
	return c.syscallConn.SyscallConn()
}

// mustAtoi parses s as a base-10 integer, failing the test if it cannot.
func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// tcpProxy forwards the connections it accepts to a fixed target address,
// keeping a handle on every proxied connection so that a test can drop them.
type tcpProxy struct {
	addr string

	mu       sync.Mutex
	conns    []net.Conn // protected by mu
	accepted int        // protected by mu, number of connections accepted
}

// newTCPProxy starts a tcpProxy forwarding to target, stopped when the test
// ends.
func newTCPProxy(t *testing.T, target string) *tcpProxy {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := &tcpProxy{addr: l.Addr().String()}
	var wg sync.WaitGroup
	wg.Go(func() {
		for {
			client, err := l.Accept()
			if err != nil {
				return
			}
			upstream, err := net.Dial("tcp", target)
			if err != nil {
				_ = client.Close()
				continue
			}
			p.mu.Lock()
			p.conns = append(p.conns, client, upstream)
			p.accepted++
			p.mu.Unlock()
			wg.Go(func() {
				_, _ = io.Copy(upstream, client)
				_ = upstream.Close()
			})
			wg.Go(func() {
				_, _ = io.Copy(client, upstream)
				_ = client.Close()
			})
		}
	})
	t.Cleanup(func() {
		_ = l.Close()
		p.dropAll()
		wg.Wait()
	})
	return p
}

// dials returns how many connections the proxy has accepted so far.
func (p *tcpProxy) dials() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.accepted
}

// dropAll closes every connection the proxy is currently forwarding.
func (p *tcpProxy) dropAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.conns {
		_ = c.Close()
	}
	p.conns = p.conns[:0]
}
