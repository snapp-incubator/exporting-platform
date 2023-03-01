FROM golang:1.18
WORKDIR /app
COPY ./ ./
RUN cd cmd && go mod download && CGO_ENABLED=0 go build -o app .

FROM alpine:3.17
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=0 /app/cmd/app ./
COPY ./configs/config.yml ./configs/config.yml
ENTRYPOINT ["/app/app"]