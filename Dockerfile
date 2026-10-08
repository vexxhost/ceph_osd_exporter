# Copyright (c) 2024 VEXXHOST, Inc.
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.27.2@sha256:5bc7f572bbaa98885a3a1fd9c0aa76b59e3e14e8628bfc316bbfd0c701e4818c AS builder
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download
COPY . /src
RUN CGO_ENABLED=0 go build -o /ceph_osd_exporter

FROM scratch
COPY --from=builder /ceph_osd_exporter /bin/ceph_osd_exporter
EXPOSE 9282
ENTRYPOINT ["/bin/ceph_osd_exporter"]
