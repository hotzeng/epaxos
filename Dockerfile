FROM golang:1.12

COPY Makefile .

COPY src/ /go/src

RUN make all

CMD ["/go/bin/server"]

EXPOSE 23333
