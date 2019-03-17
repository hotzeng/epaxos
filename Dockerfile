FROM golang:1.12
COPY Makefile .
COPY src/ /go/src
COPY VERSION .
RUN make docker

FROM alpine:latest
COPY --from=0 /go/bin/server /go/bin/client /root/
WORKDIR /root/
CMD ["/root/server"]
EXPOSE 23333 23333/udp
