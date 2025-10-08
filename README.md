# psfax

A **macOS-compatible implementation** of `ps fax`, written in Go.
It reproduces the familiar **process tree view** from Linux' `procps-ng` but uses BSD's native `/bin/ps` on macOS.

---

## 🧩 Features

- Displays a full **process tree** similar to `ps faxu`
- Highlights **executable** within each command
- Zero dependencies — single static binary

---

## ⚙️ Build

Clone and build the universal binary:

```bash
git clone https://github.com/jk779/psfax.git
cd psfax
make
```

Optionally install it system-wide:
```
sudo cp psfax /usr/local/bin/
```
----

## 💻 Usage

```bash
psfax             # show full process tree
psfax -u matz     # show only branches containing processes of user 'matz'
psfax -s iterm    # show branches containing 'iterm' in command line
psfax -p 1234     # show subtree containing PID 1234
```

Example output:
```
  PID  %CPU  %MEM       USER  COMMAND
    1   0.3   0.0       root   /sbin/launchd
  355   0.0   0.0       root  ├── /System/Library/CoreServices/...
 1345   0.0   0.1       matz  ├── /Applications/iTerm.app/Contents/MacOS/iTerm2
 ```
