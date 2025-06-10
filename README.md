# mytonprovider-backend

## how to run:

```bash
go run cmd/main.go 
```

## Dev:
vscode `launch.json` example:
```json
{
    // Use IntelliSense to learn about possible attributes.
    // Hover to view descriptions of existing attributes.
    // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Package",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd",
            // need to handle OPTIONS queries
            "buildFlags": "-tags=debug", 
            "env": {...}
        }
    ]
}
```
