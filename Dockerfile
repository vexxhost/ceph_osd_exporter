# Copyright (c) 2024 VEXXHOST, Inc.
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.27.1@sha256:f44f6e88636cfb311f9ebace870ded69d943f227bb3cb27d32ffd84ea18c43ea AS builder
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download
COPY . /src
RUN CGO_ENABLED=0 go build -o /ceph_osd_exporter

FROM scratch
COPY --from=builder /ceph_osd_exporter /bin/ceph_osd_exporter
EXPOSE 9282
ENTRYPOINT ["/bin/ceph_osd_exporter"]
