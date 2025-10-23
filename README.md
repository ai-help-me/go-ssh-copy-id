# go-ssh-copy-id

A lightweight Golang tool for distributing SSH public keys to remote hosts.  
It supports **multi-host concurrency**, **non-interactive password authentication**, and securely appends your public key to the remote `~/.ssh/authorized_keys` file.

Unlike the original ssh-copy-id, this tool does not require a local SSH client and does not rely on remote shell commands like cat, echo, or mkdir. The most important thing is the reason for the existence of this software: it does **not need to read the password from the terminal**. This is a non-interactive tool.

---

## Features

- Multi-host concurrent public key deployment (`-c` to control concurrency)  
- Password authentication via:
  - Command line (`-password PASSWORD`)
  - Password file (`--password-file FILE`)
  - Interactive prompt if neither is provided  
- Supports reading public key from:
  - File (`-i` option, default `~/.ssh/id_rsa.pub`)
  - Standard input (`stdin`)  
- Automatically creates remote `~/.ssh` directory with `0700` permissions  
- Checks for existing keys to avoid duplicates  
- **Zero remote command dependency**: works even on minimal containers or embedded systems  
- Cross-platform: Linux / macOS / Windows / ARM / x86  
- Single binary deployment, no extra dependencies  

## Install
 
```
go install github.com/ai-help-me/go-ssh-copy-id@latest
```

Use it with ansible
```
- name: test
  hosts: all
  tasks:
   - name: ssh-copy-id
     command: /usr/local/bin/go-ssh-copy-id  -i ~/.ssh/id_rsa.pub --user root --hosts {{ item }} --password {{ ansible_password }}
     with_items:
      - 172.28.25.10
      - 172.28.25.11
      - 172.28.25.12
      - 172.28.25.13
   - name: ssh connect check
     shell: ssh -o ConnectTimeout=10 -o BatchMode=yes -o StrictHostKeyChecking=no root@{{ item }} "echo 'SSH connection successful to {{ item }}'"
     with_items:
      - 172.28.25.10
      - 172.28.25.11
      - 172.28.25.12
      - 172.28.25.13
```
