package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"sync"
)

const (
	CheckSecuriry_Off int = iota
	CheckSecuriry_Queue
	CheckSecuriry_Aggressive
	CheckSecuriry_LogOnly
)

const (
	SecurityStatus_Unknown int = iota
	SecurityStatus_InProgress
	SecurityStatus_WrongCert
	SecurityStatus_HTTPS
)

type SecurityResults struct {
	sync.RWMutex
	r map[string]int
}

type Security struct {
	Title                     string
	RedirUrl                  string
	Policy                    int
	EnforceHTTPSHostnames     bool
	EnforceHTTPSVerifiedCerts bool
	CheckProxyTunnels         bool
	AllowUnknownProtocol      bool
	Results                   SecurityResults
}

func (s *Security) Redirect(id string, out chan string, input *Input, reason string) bool {
	redir_url, log_line := FormatRedirect(s.Title, s.RedirUrl, input, reason)

	if s.Policy == CheckSecuriry_LogOnly {
		ChangeLogger.Printf(log_line + " (DRY-RUN)")
		return false
	} else {
		out <- id + " OK rewrite-url=" + redir_url
		ChangeLogger.Printf(log_line)
		return true
	}
}

func (s *Security) CheckHTTPSHostnameIsIP(url URL) bool {
	ip := net.ParseIP(url.Host)
	return ip != nil
}

func (s *Security) CheckHTTPSWrongCert(url URL) bool {
	s.Results.RLock()
	check_res, ok := s.Results.r[url.Host+url.Port]
	s.Results.RUnlock()
	if !ok || check_res == SecurityStatus_Unknown {
		if s.Policy == CheckSecuriry_Aggressive {
			s.CheckHTTPSRoot(url)
			s.Results.RLock()
			check_res, _ = s.Results.r[url.Host+url.Port]
			s.Results.RUnlock()
		} else {
			ErrorLogger.Println("go check https root")
			go s.CheckHTTPSRoot(url)
			return false
		}
	}
	return check_res == SecurityStatus_WrongCert
}

func (s *Security) CheckHTTPSRoot(url URL) {
	ErrorLogger.Printf("Checking %s:%s\n", url.Host, url.Port)
	s.Results.Lock()
	s.Results.r[url.Host+url.Port] = SecurityStatus_InProgress
	s.Results.Unlock()

	tlscfg := tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionSSL30}
	tlscfg.ServerName = url.Host
	conn, err_conn := tls.Dial("tcp", fmt.Sprintf("%s:%s", url.Host, url.Port), &tlscfg)
	if err_conn != nil {
		ErrorLogger.Printf("Failed to dial %s:%s - %s\n", url.Host, url.Port, err_conn)
		s.Results.Lock()
		s.Results.r[url.Host+url.Port] = SecurityStatus_Unknown
		s.Results.Unlock()
		return
	}

	state := conn.ConnectionState()
	opts := x509.VerifyOptions{
		DNSName:       tlscfg.ServerName,
		Intermediates: x509.NewCertPool(),
	}
	for i, cert := range state.PeerCertificates {
		if i == 0 {
			continue
		}
		opts.Intermediates.AddCert(cert)
	}
	_, err_verify := state.PeerCertificates[0].Verify(opts)
	if err_verify != nil {
		s.Results.Lock()
		s.Results.r[url.Host+url.Port] = SecurityStatus_WrongCert
		s.Results.Unlock()
		ErrorLogger.Printf("%s:%s - %s\n", url.Host, url.Port, err_verify)
		return
	}

	message := fmt.Sprintf("GET / HTTP/1.1\nHost: %s\n\n", tlscfg.ServerName)
	_, err_write := io.WriteString(conn, message)
	if err_write != nil {
		ErrorLogger.Printf("Failed to write '%s' to '%s:%s' - %s", message, url.Host, url.Port, err_write)
		s.Results.Lock()
		s.Results.r[url.Host+url.Port] = SecurityStatus_Unknown
		s.Results.Unlock()
		return
	}

	reply := make([]byte, 256)
	_, err_read := conn.Read(reply)
	if err_read != nil {
		ErrorLogger.Printf("Failed to read from '%s:%s' - %s", url.Host, url.Port, err_read)
		s.Results.Lock()
		s.Results.r[url.Host+url.Port] = SecurityStatus_Unknown
		s.Results.Unlock()
		return
	}
	if bytes.HasPrefix(reply, []byte("HTTP/")) {
		s.Results.Lock()
		s.Results.r[url.Host+url.Port] = SecurityStatus_HTTPS
		s.Results.Unlock()
		return
	}
}
