# go-ssh-copy-id

A lightweight Golang tool for distributing SSH public keys to remote hosts.  
It supports **multi-host concurrency**, **password authentication**, and securely appends your public key to the remote `~/.ssh/authorized_keys` file.

Unlike the original ssh-copy-id, this tool does not require a local SSH client and does not rely on remote shell commands like cat, echo, or mkdir.

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
go github.com/ai-help-me/go-ssh-copy-id@latest
```