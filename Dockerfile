# Copyright (c) 2024 VEXXHOST, Inc.
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.27.1@sha256:1cfcdb11f37fce9429f617100f39e0251748bbaba454bd431275155701765058 AS builder
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download
COPY . /src
RUN CGO_ENABLED=0 go build -o /ceph_osd_exporter

FROM scratch
COPY --from=builder /ceph_osd_exporter /bin/ceph_osd_exporter
EXPOSE 9282
ENTRYPOINT ["/bin/ceph_osd_exporter"]
