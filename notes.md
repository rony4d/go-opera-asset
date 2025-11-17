### **Error when setting up a Debugger**

This error is occuring because I have go version 1.24 installed as well as go version 1.20 installed. My project needs go version 1.20. So what I did was to install the debugger that will work with this older go version which is delve version 1.22.1. The issue is that this debugger still points to the latest version of go and it is not compatible. So I had to find a way around it by pointing the settings.json and launch.json to the actual version of go for my current project. This problem will not exist with new projects running the latest version of go as I have tested it and everything works easily. 

Delve is picking up the Go toolchain it finds first on your system `PATH`, which is the newer Go 1.24 you installed globally. To force the debugger to use Go 1.20.14 just for this project, point Cursor/VS Code at the 1.20 toolchain.

1. **Find the Go 1.20.14 install path**

   Run the 1.20 binary directly so you know the correct `GOROOT`:

   ```bash
   /path/to/go1.20.14/bin/go env GOROOT
   ```

   (If you installed via Homebrew, it’ll be something like `/usr/local/opt/go@1.20/libexec`; adjust to whatever the command prints.)

2. **Tell the Go extension to use that toolchain**

   Create or edit `.vscode/settings.json` in the repo:

   ```json
   {
     "go.goroot": "/usr/local/opt/go@1.20/libexec",
     "go.alternateTools": {
       "go": "/usr/local/opt/go@1.20/libexec/bin/go"
     },
     "go.toolsEnvVars": {
       "GOROOT": "/usr/local/opt/go@1.20/libexec",
       "PATH": "/usr/local/opt/go@1.20/libexec/bin:${env:PATH}"
     }
   }
   ```

   Replace the paths with the value you gathered in step 1. `go.alternateTools` makes the Go extension (and Delve) invoke that binary; `go.toolsEnvVars` ensures the right `GOROOT`/`PATH` get exported when the debugger starts.

3. **Update your launch configuration**

   In `.vscode/launch.json`, add the same `env` override to the `Debug opera-asset` entry:

   ```json
   "env": {
     "GOROOT": "/usr/local/opt/go@1.20/libexec",
     "PATH": "/usr/local/opt/go@1.20/libexec/bin:${env:PATH}"
   }
   ```

   Leave `dlvToolPath` pointing at the Delve you already installed (`/Users/ugochukwu/go/bin/dlv`).

4. **Restart the Go extension ( I didn't have to do this, it just worked )** (Command Palette → “Developer: Reload Window”) and re-run the debug session.

With those overrides, Cursor will launch Delve using Go 1.20.14 even though a newer Go is present globally, eliminating the “Delve too old for Go 1.24” error.


## Running Tests

use **go test -v ./test** to run all tests and log everything