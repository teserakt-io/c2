package e4test

import (
	"crypto/rand"
	"crypto/tls"
	b64 "encoding/base64"
	"fmt"
	"go/build"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	e4 "teserakt/e4go/pkg/e4common"
	"time"
)

// findAndCheckPathFile does some light sanity checks on
// file access. If supplied an absolute file path, it checks
// we can stat this file and that it isn't stupid, like
// a directory
// if the subpath variable is a relative path this is assumed
// to be relative to one of the directories specified
// as a gopath. We search each one to try to find the
// file and return its absolute path.
func findAndCheckPathFile(subpath string) (string, error) {

	// if we
	if filepath.IsAbs(subpath) {
		fileInfo, err := os.Stat(subpath)
		if err != nil {
			return "", fmt.Errorf("Unable to stat file %s", subpath)
		}
		if fileInfo.IsDir() {
			return "", fmt.Errorf("Can't exec a directory %s", subpath)
		}
		return subpath, nil
	}

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	fullfilepath := ""

	godirs := strings.Split(gopath, ":")
	for _, godir := range godirs {
		fileTentative := filepath.Join(godir, subpath)
		fileInfo, err := os.Stat(fileTentative)
		if err == nil && !fileInfo.IsDir() {
			fullfilepath = string(fileTentative)
			break
		}
	}
	if fullfilepath == "" {
		return "", fmt.Errorf("Unable to locate file %s in any Gopath directory", subpath)
	}
	return fullfilepath, nil
}

// RunDaemon launches the specified process with arguments
// and waits for notification on an error channel to
// signal the process to exit. Should be used for testing
// daemons like the C2 backend, where the process needs to
// stay alive and we interact with it via an API.
func RunDaemon(errc chan error,
	path string,
	args []string) {

	subproc := exec.Cmd{
		Path: path,
		Args: args,
	}

	/*spstdin, _ := subproc.StdinPipe()
	spstdout, _ := subproc.StdoutPipe()
	spstderr, _ := subproc.StderrPipe()*/

	if err := subproc.Start(); err != nil {
		errc <- fmt.Errorf("runDaemon failed. %s", err)
	}

	// clean up on exit:
	defer func() {
		// send an interrupt signal to terminate the process.
		subproc.Process.Signal(os.Interrupt)
		subproc.Wait()
	}()

	// wait for error indicating an exit.
	<-errc
}

// GetRandomDBName produces a random database
// path in tmp for use with SQLite3 testing.
func GetRandomDBName() string {
	bytes := [16]byte{}
	_, err := rand.Read(bytes[:])
	if err != nil {
		panic(err)
	}
	dbCandidate := b64.StdEncoding.EncodeToString(bytes[:])
	dbCleaned1 := strings.Replace(dbCandidate, "+", "", -1)
	dbCleaned2 := strings.Replace(dbCleaned1, "/", "", -1)
	dbCleaned3 := strings.Replace(dbCleaned2, "=", "", -1)

	dbPath := fmt.Sprintf("/tmp/e4c2_unittest_%s.sqlite", dbCleaned3)
	return dbPath
}

// GenerateID generates a random ID that is e4.IDLen bytes
// in length, using a CSPRNG
func GenerateID() ([]byte, error) {
	idbytes := [e4.IDLen]byte{}
	_, err := rand.Read(idbytes[:])
	if err != nil {
		return nil, err
	}
	return idbytes[:], nil
}

// GenerateKey generates a random key
// that is e4.KeyLen bytes
// in length, using a CSPRNG
func GenerateKey() ([]byte, error) {
	keybytes := [e4.KeyLen]byte{}
	_, err := rand.Read(keybytes[:])
	if err != nil {
		return nil, err
	}
	return keybytes[:], nil
}

// ConstructHTTPSClient builds an HTTPS client for use
// with the API
func ConstructHTTPSClient() http.Client {
	tlsConfig := &tls.Config{
		//Certificates: []tls.Certificate{tlsCert},
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	httpTransport := &http.Transport{
		IdleConnTimeout: time.Second * 20, // required, or goroutines can hang.
		TLSClientConfig: tlsConfig,
		//TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	httpClient := http.Client{
		Transport: httpTransport,
	}

	return httpClient
}
