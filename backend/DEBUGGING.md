# Debugging Setup Guide

## Using Delve (dlv) - Go Debugger

### Installation

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

### Running the Server with Delve

#### Option 1: Run directly with dlv

```bash
cd backend
dlv debug cmd/server/main.go
```

Once in the debugger:
- Set breakpoint: `break ripper.go:24` (or any line number)
- Set breakpoint by function: `break ize.ProcessRipper`
- Continue: `continue` or `c`
- Step: `step` or `s`
- Next: `next` or `n`
- Print variable: `print variableName` or `p variableName`
- List code: `list`
- Exit: `exit`

#### Option 2: Attach to running process

1. Start the server normally:
```bash
cd backend
go run cmd/server/main.go
```

2. In another terminal, attach with dlv:
```bash
dlv attach $(pgrep -f "go run cmd/server/main.go")
# Or find the PID manually:
# ps aux | grep "cmd/server/main.go"
# dlv attach <PID>
```

#### Option 3: Run with headless debugger (for IDE attachment)

```bash
cd backend
dlv debug --headless --listen=:2345 --api-version=2 cmd/server/main.go
```

Then connect from your IDE (see IDE setup below).

### Setting Breakpoints

In the debugger console:
```
(dlv) break ripper.go:24
(dlv) break ripper.go:107  # Start of greedy selection loop
(dlv) break ripper.go:146  # Information gain calculation
(dlv) break ripper.go:178  # After selecting best facet value
```

Or use function names:
```
(dlv) break ize.ProcessRipper
```

## IDE Setup

### VS Code

1. Install the Go extension
2. Create `.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Server",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/backend/cmd/server",
      "env": {},
      "args": [],
      "showLog": true
    },
    {
      "name": "Attach to Server",
      "type": "go",
      "request": "attach",
      "mode": "local",
      "processId": 0
    }
  ]
}
```

3. Set breakpoints by clicking in the gutter
4. Press F5 to start debugging

### GoLand / IntelliJ IDEA

1. Go to Run → Edit Configurations
2. Click "+" → Go Build
3. Set:
   - Name: "Debug Server"
   - Run kind: "Package"
   - Package path: `ize/cmd/server`
   - Working directory: `backend`
4. Set breakpoints by clicking in the gutter
5. Click Debug button (or Shift+F9)

### Cursor / VS Code with Delve

1. Install Go extension
2. Use the launch.json above
3. Set breakpoints in `ripper.go`
4. Press F5 to start debugging

## Debugging Tips

### Viewing Variables

In Delve console:
```
(dlv) print totalItems
(dlv) print minGroupSize
(dlv) print facetValueMap
(dlv) print bestGain
(dlv) print selectedGroups
```

### Conditional Breakpoints

```
(dlv) break ripper.go:146 if iteration == 2
(dlv) break ripper.go:164 if gain > 10.0
```

### Watch Expressions

```
(dlv) print len(assignedItems)
(dlv) print totalUnassigned
(dlv) print len(selectedGroups)
```

### Step Through Information Gain Calculation

Set breakpoint at line 146, then:
```
(dlv) step    # Step into the calculation
(dlv) print p
(dlv) print t
(dlv) print P
(dlv) print T
(dlv) print ratio1
(dlv) print ratio2
(dlv) print gain
```

## Logging vs Debugging

The code now includes debug-level logging. To see logs:

```bash
# Set log level to debug (default in development)
export LOG_FORMAT=text
cd backend
go run cmd/server/main.go
```

Or check logs in the console output - debug logs will show:
- ProcessRipper start/completion
- Each iteration details
- Information gain calculations for candidates
- Selected facet values
- Final group counts

## Example Debug Session

```bash
$ cd backend
$ dlv debug cmd/server/main.go

(dlv) break ripper.go:107
Breakpoint 1 set at 0x1234567 for ize.ProcessRipper() ./internal/ize/ripper.go:107
(dlv) continue
> ize.ProcessRipper() ./internal/ize/ripper.go:107:1
=> 107:	for iteration := 0; iteration < 5; iteration++ {
(dlv) print totalItems
10
(dlv) print minGroupSize
2
(dlv) break ripper.go:164
Breakpoint 2 set at 0x1234568 for ize.ProcessRipper() ./internal/ize/ripper.go:164
(dlv) continue
> ize.ProcessRipper() ./internal/ize/ripper.go:164:1
=> 164:				if gain > bestGain || (gain == bestGain && (p > len(bestIndices) || (p == len(bestIndices) && fmt.Sprintf("%s:%s", facetName, value) < fmt.Sprintf("%s:%s", bestFacetName, bestFacetValue)))) {
(dlv) print facetName
"category"
(dlv) print value
"Electronics"
(dlv) print gain
5.234
(dlv) continue
```
