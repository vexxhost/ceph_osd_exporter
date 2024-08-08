// Copyright (c) 2024 VEXXHOST, Inc.
// SPDX-License-Identifier: Apache-2.0

package ceph

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestOsd(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Single digit",
			path:     "/var/run/ceph/ceph-osd.1.asok",
			expected: "1",
		},
		{
			name:     "Multiple digits",
			path:     "/var/run/ceph/ceph-osd.123.asok",
			expected: "123",
		},
		{
			name:     "UUID",
			path:     "/var/run/ceph/1d3e4c7b-6b3b-4b3b-8b3b-3b3b3b3b3b3b/ceph-osd.123.asok",
			expected: "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			as := AdminSocket{Path: tt.path}

			assert.Equal(t, tt.expected, as.Osd(), "unexpected OSD from AdminSocket")
		})
	}
}

func TestSendCommand(t *testing.T) {
	socketPath := "/tmp/test-ceph-osd.1.asok"
	defer os.Remove(socketPath)

	errChan := make(chan error, 1)

	go func() {
		l, err := net.Listen("unix", socketPath)
		if err != nil {
			errChan <- err
			return
		}
		defer l.Close()

		conn, err := l.Accept()
		if err != nil {
			errChan <- err
			return
		}
		defer conn.Close()

		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			errChan <- err
			return
		}

		responseBytes, err := json.Marshal(map[string]interface{}{
			"fragmentation_rating": 0.62288891320016271,
		})
		if err != nil {
			errChan <- err
			return
		}

		length := uint32(len(responseBytes))
		lenBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBuf, length)

		conn.Write(lenBuf)
		conn.Write(responseBytes)

		errChan <- nil
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	as := AdminSocket{Path: socketPath}
	command := AdminSocketCommand{
		Prefix: "bluestore allocator score block",
	}

	response, err := as.SendCommand(command)
	assert.NoError(t, err, "expected no error from SendCommand")

	expectedResponse := map[string]interface{}{
		"fragmentation_rating": 0.6228889132001627,
	}
	assert.Equal(t, expectedResponse, response, "unexpected response from SendCommand")

	if err := <-errChan; err != nil {
		t.Fatal(err)
	}
}

func TestGetAllAdminSockets(t *testing.T) {
	tests := []struct {
		name  string
		setup func() []string
	}{
		{
			name: "Multiple FSID",
			setup: func() []string {
				fsids := []string{uuid.New().String(), uuid.New().String()}
				return []string{
					fmt.Sprintf("/var/run/ceph/%s/ceph-osd.%d.asok", fsids[0], rand.Intn(1000)),
					fmt.Sprintf("/var/run/ceph/%s/ceph-osd.%d.asok", fsids[1], rand.Intn(1000)),
				}
			},
		},
		{
			name: "Single FSID",
			setup: func() []string {
				fsid := uuid.New().String()
				return []string{
					fmt.Sprintf("/var/run/ceph/%s/ceph-osd.%d.asok", fsid, rand.Intn(1000)),
				}
			},
		},
		{
			name: "Flat",
			setup: func() []string {
				return []string{
					fmt.Sprintf("/var/run/ceph/ceph-osd.%d.asok", rand.Intn(1000)),
					fmt.Sprintf("/var/run/ceph/ceph-osd.%d.asok", rand.Intn(1000)),
				}
			},
		},
		{
			name: "Mix Single FSID + Flat",
			setup: func() []string {
				fsid := uuid.New().String()
				return []string{
					fmt.Sprintf("/var/run/ceph/%s/ceph-osd.%d.asok", fsid, rand.Intn(1000)),
					fmt.Sprintf("/var/run/ceph/ceph-osd.%d.asok", rand.Intn(1000)),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			files := tt.setup()

			for _, file := range files {
				err := fs.MkdirAll(filepath.Dir(file), 0755)
				assert.NoError(t, err, "Error creating test directory: %s", filepath.Dir(file))

				err = afero.WriteFile(fs, file, []byte{}, 0644)
				assert.NoError(t, err, "Error creating test file: %s", file)
			}

			sockets, err := GetAllAdminSockets(fs)
			assert.NoError(t, err, "Error getting admin sockets")

			var socketPaths []string
			for _, socket := range sockets {
				socketPaths = append(socketPaths, socket.Path)
			}

			assert.ElementsMatch(t, files, socketPaths, "Unexpected list of sockets found")
		})
	}
}
