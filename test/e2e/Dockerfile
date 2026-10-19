ARG BUILDER_IMAGE=registry.access.redhat.com/ubi9/go-toolset:1.26@sha256:60fd6fb2998c40ad0d6561c34e83588b986a0cf7771f64e5e5c295ba87101f9b
ARG RUNTIME_IMAGE=registry.redhat.io/ubi9/ubi-minimal-pqc@sha256:b3a0b012de0c578082e1bc5b01c8bc27461b4eeb571a0b1a2b21a07dbd845a62

FROM ${BUILDER_IMAGE} AS builder
ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

COPY api/ api/
COPY assets/ assets/
COPY config/chart/ config/chart/
COPY internal/ internal/
COPY pkg/ pkg/
COPY test/ test/

RUN mkdir -p /e2e
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go test -c -o /e2e/e2e-tests ./test/e2e/
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -o /e2e/gotestsum gotest.tools/gotestsum
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -o /e2e/test2json "$(go env GOROOT)/src/cmd/test2json"
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -o /e2e/helm helm.sh/helm/v4/cmd/helm
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -o /e2e/e2e-cleanup ./test/e2e/cmd/cleanup

FROM ${RUNTIME_IMAGE} AS runner
ARG USER_ID=65532

WORKDIR /e2e
COPY --from=builder /e2e/e2e-tests .
COPY --from=builder /e2e/gotestsum .
COPY --from=builder /e2e/test2json .
COPY --from=builder /e2e/helm .
COPY --from=builder /e2e/e2e-cleanup .
COPY --from=builder /workspace/config/chart ./chart
COPY test/e2e/runner.sh ./runner.sh
RUN chmod 0755 ./runner.sh && \
    mkdir -p /artifacts/e2e-results && \
    chown -R ${USER_ID}:0 /artifacts && \
    chmod -R g=u /artifacts

USER ${USER_ID}:0

LABEL com.redhat.component="rhai-db-operator-e2e-container" \
      name="managed-open-data-hub/rhai-db-operator-e2e-rhel9" \
      summary="RHOAI database operator end-to-end test runner" \
      description="Containerized end-to-end tests for the Red Hat OpenShift AI database operator" \
      io.k8s.display-name="rhai-db-operator-e2e" \
      io.k8s.description="Containerized end-to-end tests for the Red Hat OpenShift AI database operator"

ENTRYPOINT ["/e2e/runner.sh"]
CMD []
