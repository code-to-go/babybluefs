# stratofs
Go module that provides a common interface to multiple storage interfaces.
It includes:
- a unique interface to store and retrieve data
- a synchronization mechanism to align different storage points
- a command line tool

### Supported storage options
It supports FTP, SFTP, S3, Azure Storage, HTTPS.

### Install
```
go get -u github.com/code-to-go/stratofs`
```

### Examples

#### Command line
```
# list the content of a storage named myS3
stratofs list myS3

# create a storage of type SFTP
stratofs create sftp
 
```

#### Get the first file from S3
```go
c := fs.Config{
    Name:  "My S3",
    Group: "public",
    S3: fs.S3Config{
        endpoint, 
        bucket, 
        Location, 
        accessKey, 
        secret},
}
 
 f, err := fs.NewFS(c)
 ls, err := f.ReadDir("/", 0)
 if len(ls) > 0 {
    buf := bytes.NewBuffer(nil)
	ls.Pull(ls[0].Name(), buf)	 
 }
 
```

#### Upload a file from local file system into SFTP server
```go
c := fs.Config{
    Name:       "My SFTP",
    Group:      "public",
    SFTP:       fs.SFTPConfig{
        addr,
        username,
        password,
        base,
    },
}
remote, err := fs.NewFS(c)
local := fs.NewLocal("~/Documents", 0644) //NewFS would work too)

fs.Copy(local, remote, "file.txt", "file.txt", false, 0)
```

#### Set metadata attributes for a file
```go
f, err := fs.NewFS(c)
meta := MyFancyWhateverStruct {
field1: ...,
field2: ...,
}
fs.SetMeta(f, "file.txt", &meta)
fs.GetMeta(f, "file.txt", &meta)

}


```
