FROM golang:1.16-alpine AS build

WORKDIR /kubescout
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bin/kubescout .

FROM alpine:3 AS runtime
COPY --from=build /kubescout/bin/kubescout /usr/local/bin/
RUN chmod +x /usr/local/bin/kubescout

LABEL url=https://github.com/reallyliri/KubeScout
ENTRYPOINT ["kubescout"]
