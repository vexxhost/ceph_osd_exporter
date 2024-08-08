// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package ceph

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"strings"

	"github.com/spf13/afero"
)

type AdminSocketCommand struct {
	Prefix string `json:"prefix"`
	Format string `json:"format,omitempty"`
}

type AdminSocket struct {
	Path string
}

func (as *AdminSocket) Osd() string {
	parts := strings.Split(as.Path, ".")
	return parts[len(parts)-2]
}

func (as *AdminSocket) SendCommand(command AdminSocketCommand) (map[string]interface{}, error) {
	conn, err := net.Dial("unix", as.Path)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	commandBytes, err := json.Marshal(command)
	if err != nil {
		return nil, err
	}

	_, err = conn.Write(append(commandBytes, 0))
	if err != nil {
		return nil, err
	}

	lenBuf := make([]byte, 4)
	_, err = conn.Read(lenBuf)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)

	response := make([]byte, length)
	_, err = conn.Read(response)
	if err != nil {
		return nil, err
	}

	var responseMap map[string]interface{}
	err = json.Unmarshal(response, &responseMap)
	if err != nil {
		return nil, err
	}

	return responseMap, nil
}

func GetAllAdminSockets(filesystem afero.Fs) ([]AdminSocket, error) {
	var sockets []AdminSocket

	err := afero.Walk(filesystem, "/var/run/ceph", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(info.Name(), "ceph-osd.") && strings.HasSuffix(info.Name(), ".asok") {
			sockets = append(sockets, AdminSocket{Path: path})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan admin sockets: %w", err)
	}

	return sockets, nil
}
