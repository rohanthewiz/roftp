package roftp

import (
	"fmt"

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
		fmt.Println("Attempting to Login...")
	}
	return conn.Login(opts.User, opts.Word)
}
