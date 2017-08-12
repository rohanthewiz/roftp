package roftp

import (
	"os"
	"github.com/jlaffaye/ftp"
	"github.com/rohanthewiz/serr"
	"path/filepath"
)

type FTPOptions struct {
	User string
	Word string
	Server string
	Port string
}

type FileData struct {
	Name string
	Size uint64
}

// The first step in using the roftp package is to get a new logged in connection,
// and cache it locally
func NewFTPConn(opts FTPOptions) (*ftp.ServerConn, error) {
	println("Attempting ftp connection")
	conn, err := ftp.Connect(opts.Server + ":" + opts.Port)
	if err != nil {
		return nil, serr.Wrap(err, "Error connecting to FTP Server")
	}
	println("FTP basic connection established. We still need to login")
	err = Login(conn, opts)
	if err != nil {
		return nil, serr.Wrap(err, "Error logging in to ftp server")
	}
	return conn, nil
}

// Login on the supplied basic connection
func Login(conn *ftp.ServerConn, opts FTPOptions) error {
	println("Attempting to Login...")
	return conn.Login(opts.User, opts.Word)
}

// Change to the serverPath directory and List files
// Provide an already logged in connection
func ListFiles(conn *ftp.ServerConn, serverPath string) (filesData []FileData, err error) {
	if err = conn.ChangeDir(serverPath); err != nil {
		return filesData, serr.Wrap(err, "Error changing directory on ftp server")
	}
	curr_path, err := conn.CurrentDir()
	if err != nil {
		return filesData, serr.Wrap(err, "Unable to obtain current directory")
	}
	println("Current path:", curr_path)

	entries, err := conn.List("")
	if  err != nil {
		return nil, serr.Wrap(err, "Error listing files")
	}
	println(len(entries), "file(s) found at", curr_path)
	for _, entry := range entries {
		filesData = append(filesData, FileData{ entry.Name, entry.Size })
		//fmt.Printf("%s\tsize: %d\n", entry.Name, entry.Size)
	}
	return
}

// Upload file to the server
// conn should be already logged in and current directory changed to desired dir on server
// ListFiles will change directory
func UploadFile(conn *ftp.ServerConn, fullPath, serverPath string, destOpt ...string) error {
	file, err := os.Open(fullPath)
	if err != nil {
		return serr.Wrap(err, "Unable to open file for upload")
	}
	defer file.Close()

	if len(destOpt) > 0 {
		serverPath = filepath.Join(serverPath, destOpt[0])
	}

	// Upload
	println("Uploading sermon:", fullPath)
	err = conn.Stor(serverPath, file)
	if err != nil {
		return serr.Wrap(err, "Error uploading file", "actual_server_path", serverPath)
	}

	return err
}
