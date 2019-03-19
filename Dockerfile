FROM golang:1.12
COPY Makefile .
COPY src/ /go/src
COPY VERSION .
RUN make docker

FROM alpine:latest
COPY --from=0 /go/bin/server /go/bin/client /root/
COPY entrypoint.sh VERSION /root/
WORKDIR /root/
ENTRYPOINT ["/root/entrypoint.sh"]
CMD ["/root/server"]
EXPOSE 23330/udp
