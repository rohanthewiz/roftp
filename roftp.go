package roftp

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jlaffaye/ftp"
	"github.com/rohanthewiz/rerr"
)

type FTPOptions struct {
	User    string
	Word    string
	Server  string
	Port    string
	Verbose bool
}

type FileData struct {
	Name string
	Size uint64
	Type string
}

// FTPConn wraps ftp.ServerConn so we don't have to expose the core lib
type FTPConn struct {
	Conn *ftp.ServerConn
}

// NewFTPConn The first step in using the roftp package is to get a new logged in connection,
// and cache it locally
func NewFTPConn(opts FTPOptions) (fcon FTPConn, err error) {
	if opts.Verbose {
		println("Attempting ftp connection...")
		fmt.Printf("**** FTP Options ->%#v\n", opts)
	}

	conn, err := ftp.Connect(opts.Server + ":" + opts.Port)
	if err != nil {
		return fcon, rerr.Wrap(err, "Error connecting to FTP Server")
	}
	if opts.Verbose {
		println("FTP basic connection established. We still need to login")
	}

	err = login(conn, opts)
	if err != nil {
		return fcon, rerr.Wrap(err, "Error logging in to ftp server")
	}
	return FTPConn{Conn: conn}, nil
}

// login on the supplied basic connection
func login(conn *ftp.ServerConn, opts FTPOptions) error {
	if opts.Verbose {
		println("Attempting to Login...")
	}
	return conn.Login(opts.User, opts.Word)
}

// Change to the serverPath directory and List files
// Provide an already logged in connection
// ListFiles will change directory to the listed directory
func ListFiles(fcon FTPConn, serverPath string) (filesData []FileData, err error) {
	if fcon.Conn == nil {
		return filesData, rerr.New("Provided ftp connection is invalid. Did you use the NewFTPConn() function?")
	}

	currPath, err := ChDir(fcon, serverPath)
	if err != nil {
		return filesData, rerr.Wrap(err, "Unable to change current dir")
	}
	println("Current path:", currPath)

	entries, err := fcon.Conn.List("")
	if err != nil {
		return nil, rerr.Wrap(err, "Error listing files", "currentDir", currPath)
	}
	println(len(entries), "item(s) found at", currPath)
	for _, entry := range entries {
		fileType := "other"
		switch entry.Type {
		case ftp.EntryTypeFile:
			fileType = "file"
		case ftp.EntryTypeFolder:
			fileType = "directory"
		}
		filesData = append(filesData, FileData{Name: entry.Name, Size: entry.Size, Type: fileType})
	}
	return
}

// Change directory and return the new path or err
func ChDir(fcon FTPConn, serverPath string) (currPath string, err error) {
	if fcon.Conn == nil {
		return currPath, rerr.New("Provided ftp connection is invalid. Did you use the NewFTPConn() function?")
	}
	if err = fcon.Conn.ChangeDir(serverPath); err != nil {
		return currPath, rerr.Wrap(err, "Error changing directory on ftp server", "requestedPath", serverPath)
	}
	currPath, err = fcon.Conn.CurrentDir()
	if err != nil {
		return currPath, rerr.Wrap(err, "Unable to obtain server's current directory after changing directory")
	}
	return
}

// Upload file to the server
// conn should be already logged in and current directory changed to desired dir on server
// Server path is dest path (without filename) on server
func UploadFile(fcon FTPConn, srcFullPath, serverPath string, destFilename ...string) error {
	if fcon.Conn == nil {
		return rerr.New("Provided ftp connection is invalid. Did you use the NewFTPConn() function?")
	}
	file, err := os.Open(srcFullPath)
	if err != nil {
		return rerr.Wrap(err, "Unable to open file for upload")
	}
	defer func() {
		_ = file.Close()
	}()

	if len(destFilename) > 0 {
		serverPath = filepath.Join(serverPath, destFilename[0])
	}

	// Upload
	println("Uploading:", srcFullPath) // TODO Show srcFullPath ==> serverPath + destFilename
	err = fcon.Conn.Stor(serverPath, file)
	if err != nil {
		return rerr.Wrap(err, "Error uploading file", "srcFile", srcFullPath, "actual_server_path", serverPath)
	}
	println("Upload of", srcFullPath, "completed")
	return err
}

// Download and write file from server
func DownloadFiles(fcon FTPConn, serverPath string, limit ...int) (success int, fail int, err error) {
	if fcon.Conn == nil {
		return success, fail, rerr.New("Provided ftp connection is invalid. Did you use the NewFTPConn() function?")
	}
	items, err := ListFiles(fcon, serverPath)
	if err != nil {
		return 0, 0, rerr.Wrap(err, "Could not obtain dir entries by name")
	}

	lmt := 0
	if len(limit) > 0 {
		lmt = limit[0]
	}
	println("The download limit is", lmt)
	for _, item := range items {
		if lmt != 0 && success+1 > lmt {
			break
		}
		if item.Type != "file" || item.Name == "." || item.Name == ".." {
			continue
		}
		println("Downloading", item.Name)
		err = DownloadFile(fcon, serverPath, item.Name)
		if err != nil {
			fail++
			println(err.Error())
			continue
		}
		success++
	}
	return
}
func DownloadFile(fcon FTPConn, serverPath, destName string) error {
	if fcon.Conn == nil {
		return rerr.New("Provided ftp connection is invalid. Did you use the NewFTPConn() function?")
	}
	data, err := DownloadFileBuffer(fcon, filepath.Join(serverPath, destName))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(destName, data, 0664)
	if err != nil {
		return rerr.Wrap(err)
	}
	return nil
}

// Download file from server as []byte
func DownloadFileBuffer(fcon FTPConn, serverPath string) (fileData []byte, err error) {
	if fcon.Conn == nil {
		return fileData, rerr.New("Provided ftp connection is invalid. Did you use the NewFTPConn() function?")
	}
	println("Downloading", serverPath, "to buffer")
	resp, err := fcon.Conn.Retr(serverPath)
	if err != nil {
		return nil, rerr.Wrap(err, "Error downloading file from server", "remote_file", serverPath)
	}
	// resp.SetDeadline(time.Now().Add(time.Minute * 15))
	fileData, err = ioutil.ReadAll(resp)
	if err != nil {
		return nil, rerr.Wrap(err, "Error reading file from server", "remote_file", serverPath)
	}
	println(len(fileData), "bytes read")
	_ = resp.Close()
	return
}
