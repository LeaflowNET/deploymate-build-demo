FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod main.go ./
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" -o /demo .

FROM scratch
COPY --from=build /demo /demo
USER 65532:65532
ENV PORT=8080
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 CMD ["/demo", "-healthcheck"]
ENTRYPOINT ["/demo"]
