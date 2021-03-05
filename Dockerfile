FROM golang:latest AS builder
LABEL stage=intermediate
COPY . /
WORKDIR /
ENV GO111MODULE=on
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /main .

FROM openjdk:8u212-jre-alpine
LABEL maintainer="Hendrik Jonas Schlehlein <hendrik.schlehlein@gmail.com>"
RUN apk --no-cache add ca-certificates
WORKDIR /
RUN mkdir configs
COPY --from=builder main ./
RUN echo "eula=$MC_EULA" > eula.txt
RUN chmod +x ./main
ENTRYPOINT [ "./main" ]
