# DistributedTextEditor

CRDT Implementation of a Collaborative Text Editor done in GoLang.

## Usage
```bash
go build editor.go
```
 
 ### Windows
 ```bash
./editor.exe
```

 ### Mac
 ```bash
chmod u+x editor
./editor
```

### Windows & Mac
Go to localhost:8080 after running the windows or mac specific execution command.
Multiple tabs, windows, or browsers can be opened to create multiple clients.
Modifications made on the document in one view will be reflected in the other views.
