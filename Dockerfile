FROM node:22-bookworm-slim AS frontend
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod *.go ./
COPY --from=frontend /web/dist ./web/dist
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" -o /demo . && mkdir -p /data

FROM scratch
COPY --from=build /demo /demo
COPY --from=build --chown=65532:65532 /data /data
USER 65532:65532
ENV PORT=8080 DATA_DIR=/data
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 CMD ["/demo", "-healthcheck"]
ENTRYPOINT ["/demo"]
