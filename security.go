package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
)

const (
	CheckSecuriry_Off int = iota
	CheckSecuriry_Queue
	CheckSecuriry_Aggressive
	CheckSecuriry_LogOnly
)

const (
	SecurityStatus_Unknown int = iota
	SecurityStatus_WrongCert
	SecurityStatus_HTTPS
)

type Security struct {
	Title                     string
	RedirUrl                  string
	Policy                    int
	EnforceHTTPSHostnames     bool
	EnforceHTTPSVerifiedCerts bool
	CheckProxyTunnels         bool
	AllowUnknownProtocol      bool
	Results                   map[string]int
}

func (s *Security) Redirect(id string, out chan string, input *Input, reason string) {
	redir_url, log_line := FormatRedirect(s.Title, s.RedirUrl, input, reason)

	if s.Policy == CheckSecuriry_LogOnly {
		Pass(id, out, reason)
		ChangeLogger.Printf(log_line + " (DRY-RUN)")
	} else {
		out <- id + " OK rewrite-url=" + redir_url
		ChangeLogger.Printf(log_line)
	}
}

func (s *Security) CheckHTTPSHostnameIsIP(url URL) bool {
	if config.Security.EnforceHTTPSHostnames == true {
		ip := net.ParseIP(url.Host)
		return ip != nil
	}
	return false
}

func (s *Security) CheckHTTPSWrongCert(url URL) bool {
	s.CheckHTTPSRoot(url)
	return s.Results[url.Host+url.Port] == SecurityStatus_WrongCert
}

func (s *Security) CheckHTTPSRoot(url URL) {
	s.Results[url.Host+url.Port] = SecurityStatus_Unknown

	tlscfg := tls.Config{InsecureSkipVerify: true}
	tlscfg.ServerName = url.Host
	conn, err_conn := tls.Dial("tcp", fmt.Sprintf("%s:%s", url.Host, url.Port), &tlscfg)
	if err_conn != nil {
		ErrorLogger.Printf("Failed to dial %s:%s - %s\n", url.Host, url.Port, err_conn)
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
		s.Results[url.Host+url.Port] = SecurityStatus_WrongCert
		ErrorLogger.Printf("%s:%s - %s\n", url.Host, url.Port, err_verify)
		return
	}

	message := fmt.Sprintf("GET / HTTP/1.1\nHost: %s\n\n", tlscfg.ServerName)
	_, err_write := io.WriteString(conn, message)
	if err_write != nil {
		ErrorLogger.Printf("Failed to write '%s' to '%s:%s' - %s", message, url.Host, url.Port, err_write)
		return
	}

	reply := make([]byte, 256)
	_, err_read := conn.Read(reply)
	if err_read != nil {
		ErrorLogger.Printf("Failed to read from '%s:%s' - %s", url.Host, url.Port, err_read)
		return
	}
	if bytes.HasPrefix(reply, []byte("HTTP/")) {
		s.Results[url.Host+url.Port] = SecurityStatus_HTTPS
		return
	}
}
