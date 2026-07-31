# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    GOARM=${TARGETVARIANT#v} \
    go build -trimpath -ldflags="-s -w" -o /simple-time-service .

FROM scratch

COPY --from=build /simple-time-service /simple-time-service
USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/simple-time-service"]
