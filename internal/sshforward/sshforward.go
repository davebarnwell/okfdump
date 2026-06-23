package sshforward

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type Config struct {
	Host                  string
	Port                  int
	User                  string
	Password              string
	PasswordEnv           string
	KeyPath               string
	KeyPassphrase         string
	KnownHostsPath        string
	InsecureIgnoreHostKey bool
	TargetHost            string
	TargetPort            int
}

type Tunnel struct {
	listener net.Listener
	client   *ssh.Client
	done     chan struct{}
	once     sync.Once
}

func Start(ctx context.Context, cfg Config) (*Tunnel, error) {
	auth, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}
	if len(auth) == 0 {
		return nil, errors.New("SSH password or private key is required")
	}

	hostKeyCallback, err := hostKeyCallback(cfg)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            auth,
		HostKeyCallback: hostKeyCallback,
		Timeout:         15 * time.Second,
	}

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)))
	if err != nil {
		return nil, err
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)), sshConfig)
	if err != nil {
		conn.Close()
		return nil, err
	}
	client := ssh.NewClient(sshConn, chans, reqs)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		client.Close()
		return nil, err
	}

	tunnel := &Tunnel{listener: listener, client: client, done: make(chan struct{})}
	go tunnel.acceptLoop(cfg.TargetHost, cfg.TargetPort)
	return tunnel, nil
}

func (t *Tunnel) LocalHost() string {
	host, _, _ := net.SplitHostPort(t.listener.Addr().String())
	return host
}

func (t *Tunnel) LocalPort() int {
	_, port, _ := net.SplitHostPort(t.listener.Addr().String())
	value, _ := strconv.Atoi(port)
	return value
}

func (t *Tunnel) Close() error {
	var err error
	t.once.Do(func() {
		close(t.done)
		err = t.listener.Close()
		if closeErr := t.client.Close(); err == nil {
			err = closeErr
		}
	})
	return err
}

func (t *Tunnel) acceptLoop(targetHost string, targetPort int) {
	for {
		local, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.done:
				return
			default:
				continue
			}
		}
		go t.forward(local, net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
	}
}

func (t *Tunnel) forward(local net.Conn, target string) {
	remote, err := t.client.Dial("tcp", target)
	if err != nil {
		local.Close()
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go copyAndClose(&wg, remote, local)
	go copyAndClose(&wg, local, remote)
	wg.Wait()
}

func copyAndClose(wg *sync.WaitGroup, dst net.Conn, src net.Conn) {
	defer wg.Done()
	_, _ = io.Copy(dst, src)
	_ = dst.Close()
	_ = src.Close()
}

func authMethods(cfg Config) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	if cfg.Password != "" {
		methods = append(methods, ssh.Password(cfg.Password))
	}
	if cfg.KeyPath != "" {
		signer, err := readSigner(cfg.KeyPath, cfg.KeyPassphrase)
		if err != nil {
			return nil, err
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	return methods, nil
}

func readSigner(path string, passphrase string) (ssh.Signer, error) {
	expanded, err := expandPath(path)
	if err != nil {
		return nil, err
	}

	key, err := os.ReadFile(expanded)
	if err != nil {
		return nil, fmt.Errorf("read SSH key: %w", err)
	}
	if passphrase != "" {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("parse SSH key: %w", err)
		}
		return signer, nil
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parse SSH key: %w", err)
	}
	return signer, nil
}

func hostKeyCallback(cfg Config) (ssh.HostKeyCallback, error) {
	if cfg.InsecureIgnoreHostKey {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	path, err := expandPath(cfg.KnownHostsPath)
	if err != nil {
		return nil, err
	}
	callback, err := knownHostsCallback(path)
	if err != nil {
		return nil, err
	}
	return callback, nil
}

func knownHostsCallback(path string) (ssh.HostKeyCallback, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("read SSH known_hosts: %w; pass --ssh-insecure-ignore-host-key to bypass verification", err)
	}
	callback, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("parse SSH known_hosts: %w", err)
	}
	return callback, nil
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}
