package bosh

import (
	"fmt"
	"io"
	"net"

	"github.com/EngineerBetter/concourse-up/config"
	"github.com/EngineerBetter/concourse-up/director"
	"github.com/EngineerBetter/concourse-up/iaas"
	"github.com/EngineerBetter/concourse-up/terraform"
	"github.com/lib/pq"
	"golang.org/x/crypto/ssh"
)

//GCPClient is an GCP specific implementation of IClient
type GCPClient struct {
	config   config.Config
	metadata terraform.IAASMetadata
	director director.IClient
	db       Opener
	stdout   io.Writer
	stderr   io.Writer
	provider iaas.Provider
}

//NewGCPClient returns a GCP specific implementation of IClient
func NewGCPClient(config config.Config, metadata terraform.IAASMetadata, director director.IClient, stdout, stderr io.Writer, provider iaas.Provider) (IClient, error) {
	directorPublicIP, err := metadata.Get("DirectorPublicIP")
	if err != nil {
		return nil, err
	}
	addr := net.JoinHostPort(directorPublicIP, "22")
	key, err := ssh.ParsePrivateKey([]byte(config.PrivateKey))
	if err != nil {
		return nil, err
	}
	conf := &ssh.ClientConfig{
		User:            "vcap",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(key)},
	}
	var boshDBAddress, boshDBPort string

	boshDBAddress, err = metadata.Get("BoshDBAddress")
	if err != nil {
		return nil, err
	}
	boshDBPort, err = metadata.Get("BoshDBPort")
	if err != nil {
		return nil, err
	}

	db, err := newProxyOpener(addr, conf, &pq.Driver{},
		fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
			config.RDSUsername,
			config.RDSPassword,
			boshDBAddress,
			boshDBPort,
			config.RDSDefaultDatabaseName,
		),
	)
	if err != nil {
		return nil, err
	}
	return &GCPClient{
		config:   config,
		metadata: metadata,
		director: director,
		db:       db,
		stdout:   stdout,
		stderr:   stderr,
		provider: provider,
	}, nil
}

//Cleanup is GCP specific implementation of Cleanup
func (client *GCPClient) Cleanup() error {
	return client.director.Cleanup()
}
