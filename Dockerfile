# Start by building the application.
FROM golang:1.24 as build

WORKDIR /go/src/app
COPY . ./
COPY static static

RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/bin/app cmd/main.go

# Now copy it into our base image.
FROM gcr.io/distroless/static-debian11
COPY --from=build /go/bin/app /
VOLUME /db
EXPOSE 80
ENTRYPOINT ["/app", "-addr", ":80", "-dburl", "file:/db/chores.sqlite?cache=shared"]
