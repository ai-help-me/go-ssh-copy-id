package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type HostTask struct {
	Host     string
	User     string
	Password string
	Port     int
	PubKey   string
	Timeout  time.Duration
}

func main() {
	var (
		hostsStr     string
		user         string
		keyPath      string
		cmdPassword  string
		passwordFile string
		port         int
		concurrency  int
		timeout      time.Duration
	)

	flag.StringVar(&hostsStr, "hosts", "", "Comma-separated list of target hosts")
	flag.StringVar(&user, "user", "root", "SSH login username")
	flag.StringVar(&keyPath, "i", filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa.pub"), "Public key file path (default ~/.ssh/id_rsa.pub)")
	flag.StringVar(&cmdPassword, "password", "", "SSH password specified in command line (less secure)")
	flag.StringVar(&passwordFile, "password-file", "", "File containing password to avoid cleartext input")
	flag.IntVar(&concurrency, "c", 5, "Number of concurrent SSH connections (default 5)")
	flag.IntVar(&port, "port", 22, "SSH port (default 22)")
	flag.DurationVar(&timeout, "timeout", 10*time.Second, "SSH connection timeout")
	flag.Parse()

	if hostsStr == "" {
		fmt.Println("Please specify target hosts using -hosts, e.g. -hosts \"192.168.1.10,192.168.1.11\"")
		os.Exit(1)
	}
	hosts := strings.Split(hostsStr, ",")

	pubKey, err := readPublicKey(keyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read public key: %v\n", err)
		os.Exit(1)
	}

	passwd, err := readPassword(cmdPassword, passwordFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read password: %v\n", err)
		os.Exit(1)
	}

	tasks := make(chan string, len(hosts))
	for _, h := range hosts {
		tasks <- strings.TrimSpace(h)
	}
	close(tasks)

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for host := range tasks {
				sem <- struct{}{}
				err := copyKeyToHost(&HostTask{
					Host:     host,
					User:     user,
					Password: passwd,
					Port:     port,
					PubKey:   pubKey,
					Timeout:  timeout,
				})
				<-sem

				if err != nil {
					fmt.Printf("[%s] ERROR: %v\n", host, err)
				} else {
					fmt.Printf("[%s] Public key installed successfully\n", host)
				}
			}
		}()
	}

	wg.Wait()
}

// readPublicKey reads the public key from file
func readPublicKey(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(homeDir, path[1:])
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", fmt.Errorf("empty public key")
	}
	return content, nil
}

// readPassword reads the password from command line, file, or interactively
func readPassword(cmdlinePass, file string) (string, error) {
	if cmdlinePass != "" {
		return cmdlinePass, nil
	}
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	fmt.Print("Enter SSH password: ")
	reader := bufio.NewReader(os.Stdin)
	pass, _ := reader.ReadString('\n')
	return strings.TrimSpace(pass), nil
}

// copyKeyToHost copies the public key to the remote host using SFTP
func copyKeyToHost(task *HostTask) error {
	addr := fmt.Sprintf("%s:%d", task.Host, task.Port)
	config := &ssh.ClientConfig{
		User:            task.User,
		Auth:            []ssh.AuthMethod{ssh.Password(task.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         task.Timeout,
	}

	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}
	defer conn.Close()

	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %v", err)
	}
	defer sftpClient.Close()

	sshDir := ".ssh"
	authFile := ".ssh/authorized_keys"

	// Create ~/.ssh directory if not exists
	if err := sftpClient.MkdirAll(sshDir); err != nil {
		return fmt.Errorf("failed to create %s directory: %v", sshDir, err)
	}
	sftpClient.Chmod(sshDir, 0700)

	// Open or create authorized_keys file
	f, err := sftpClient.OpenFile(authFile, os.O_RDWR|os.O_CREATE|os.O_APPEND)
	if err != nil {
		return fmt.Errorf("failed to open %s: %v", authFile, err)
	}
	defer f.Close()

	// Read existing content
	existing, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", authFile, err)
	}

	if bytes.Contains(existing, []byte(task.PubKey)) {
		return nil // Key already exists
	}

	// Append public key
	if _, err := f.Write([]byte(task.PubKey + "\n")); err != nil {
		return fmt.Errorf("failed to append public key: %v", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to write public key to remote disk: %v", err)
	}
	// Set file permission
	if err := sftpClient.Chmod(authFile, 0600); err != nil {
		return fmt.Errorf("failed to set file permission: %v", err)
	}

	return nil
}
