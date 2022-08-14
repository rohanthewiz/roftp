package roftp

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jlaffaye/ftp"
	"github.com/rohanthewiz/rerr"
)

const errMsgConnInit = "ftp connection is not initialized Did you use the NewFTPConn() function?"

// Quit quits the FTP session
// Perhaps we should defer Quit() after obtaining a successfully initialized FTPConn
func (fc FTPConn) Quit() (err error) {
	if fc.Conn == nil {
		return rerr.New(errMsgConnInit)
	}
	return fc.Conn.Quit()
}

// ListFiles will change directory to the given directory and List files
func (fc FTPConn) ListFiles(serverPath string) (filesData []FileData, err error) {
	if fc.Conn == nil {
		return filesData, rerr.New(errMsgConnInit)
	}

	currPath, err := fc.ChDir(serverPath)
	if err != nil {
		return filesData, rerr.Wrap(err, "Unable to change current dir")
	}
	fmt.Println("Current path:", currPath)

	entries, err := fc.Conn.List("")
	if err != nil {
		return nil, rerr.Wrap(err, "Error listing files", "currentDir", currPath)
	}
	fmt.Println(len(entries), "item(s) found at", currPath)

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

// ChDir changes directory and return the new path or err
func (fc FTPConn) ChDir(serverPath string) (currPath string, err error) {
	if fc.Conn == nil {
		return currPath, rerr.New(errMsgConnInit)
	}
	if err = fc.Conn.ChangeDir(serverPath); err != nil {
		return currPath, rerr.Wrap(err, "Error changing directory on ftp server", "requestedPath", serverPath)
	}
	currPath, err = fc.Conn.CurrentDir()
	if err != nil {
		return currPath, rerr.Wrap(err, "Unable to obtain server's current directory after changing directory")
	}
	return
}

// UploadFile uploads a file to the server
// srcFullPath is the source file spec (path and filename)
// Server path is dest path (without filename) on server
// destFilename is optional - default is to use the source file name
func (fc FTPConn) UploadFile(srcFullPath, serverPath string, destFilename ...string) error {
	if fc.Conn == nil {
		return rerr.New(errMsgConnInit)
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
	fmt.Println("Uploading:", srcFullPath)
	err = fc.Conn.Stor(serverPath, file)
	if err != nil {
		return rerr.Wrap(err, "Error uploading file", "srcFile", srcFullPath, "actual_server_path", serverPath)
	}
	fmt.Println("Upload of", srcFullPath, "to", serverPath, "completed")
	return err
}

// DownloadFiles Downloads and write files from server
func (fc FTPConn) DownloadFiles(serverPath string, limit ...int) (successes int, fails int, err error) {
	if fc.Conn == nil {
		return successes, fails, rerr.New(errMsgConnInit)
	}

	items, err := fc.ListFiles(serverPath)
	if err != nil {
		return 0, 0, rerr.Wrap(err, "Could not obtain dir entries by name")
	}

	lmt := 0
	if len(limit) > 0 {
		lmt = limit[0]
	}

	fmt.Println("The download limit is", lmt)
	for _, item := range items {
		if lmt != 0 && successes+1 > lmt {
			break
		}
		if item.Type != "file" || item.Name == "." || item.Name == ".." {
			continue
		}
		fmt.Println("Downloading", item.Name)
		err = fc.DownloadFile(serverPath, item.Name)
		if err != nil {
			fails++
			fmt.Println(err.Error())
			continue
		}
		successes++
	}
	return
}

func (fc FTPConn) DownloadFile(serverPath, destName string) error {
	if fc.Conn == nil {
		return rerr.New(errMsgConnInit)
	}
	data, err := fc.DownloadFileToBuffer(filepath.Join(serverPath, destName))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(destName, data, 0664)
	if err != nil {
		return rerr.Wrap(err)
	}
	return nil
}

// DownloadFileToBuffer downloads a file given as a full path from server as []byte
func (fc FTPConn) DownloadFileToBuffer(remoteFileSpec string) (fileData []byte, err error) {
	if fc.Conn == nil {
		return fileData, rerr.New(errMsgConnInit)
	}

	fmt.Println("Downloading", remoteFileSpec, "to buffer")
	resp, err := fc.Conn.Retr(remoteFileSpec)
	if err != nil {
		return nil, rerr.Wrap(err, "Error downloading file from server", "remote_file", remoteFileSpec)
	}
	defer func() { _ = resp.Close() }()

	// resp.SetDeadline(time.Now().Add(time.Minute * 15))
	fileData, err = ioutil.ReadAll(resp)
	if err != nil {
		return nil, rerr.Wrap(err, "Error reading file from server", "remote_file", remoteFileSpec)
	}

	fmt.Println(len(fileData), "bytes read")
	_ = resp.Close()
	return
}
