# Files-Sync-S3
Lil util tool that syncs files in a folder between computers. The files are small and stored in the S3 bucket.
Requires the bucket to be public.

### Build and Run

```bash
go mod download
go build -o ./bin/foldersync.exe ./src/main.go && ./bin/foldersync.exe
```

or

```bash
make run
```
