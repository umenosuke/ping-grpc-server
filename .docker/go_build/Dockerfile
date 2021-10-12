FROM centos:7 AS builder

RUN yum update -y
RUN yum install -y wget zip unzip git

ARG GO_VERSION='1.17.2'
ARG GO_BINARY='go'"${GO_VERSION}"'.linux-amd64.tar.gz'
RUN wget https://dl.google.com/go/"${GO_BINARY}" -O /tmp/"${GO_BINARY}"
RUN tar -C /usr/local -xzf /tmp/"${GO_BINARY}"
ENV GOROOT=/usr/local/go
RUN echo 'export GOROOT="'"${GOROOT}"'"' >> /etc/profile.d/myGo.sh
ENV GOPATH=/usr/local/gopath
RUN mkdir -p ${GOPATH} && chmod 777 ${GOPATH}
RUN echo 'export GOPATH="'"${GOPATH}"'"' >> /etc/profile.d/myGo.sh
RUN echo 'export PATH=${PATH}:${GOROOT}/bin:${GOPATH}/bin' >> /etc/profile.d/myGo.sh
ENV GOCACHE=/.cache/go-build
RUN mkdir -p ${GOCACHE} && chmod 777 ${GOCACHE}
ENV PATH=${PATH}:${GOROOT}/bin:${GOPATH}/bin

ENV TZ=Asia/Tokyo


FROM builder AS build

COPY ./.script/ /target_data/.script
COPY ./go.mod /target_data/go.mod
COPY ./go.sum /target_data/go.sum
COPY ./src/ /target_data/src
COPY ./proto/ /target_data/proto

ARG _GIT_TAG
ARG _GIT_HASH

WORKDIR /target_data
RUN .script/go_build.sh 'linux' 'amd64' './src' "build/ping-grpc-server"


FROM alpine:latest AS release
COPY --from=build /target_data/build/ping-grpc-server .
EXPOSE 5555
ENTRYPOINT ["/ping-grpc-server"]
STOPSIGNAL SIGINT
