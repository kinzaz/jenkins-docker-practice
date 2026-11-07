FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
ARG TARGETOS
ARG TARGETARCH
ARG RELEASE=development
RUN CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
    go build -trimpath \
    -ldflags="-X main.version=$RELEASE" \
    -o /todo-api . \

FROM scratch
COPY --from=build /todo-api /todo-api
USER 65532:65532
ENTRYPOINT ["/todo-api"]
