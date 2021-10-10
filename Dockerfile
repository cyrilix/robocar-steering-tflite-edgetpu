FROM docker.io/golang:1.17 as gobuilder

FROM docker.io/cyrilix/tflite-builder:v2.6.0

COPY --from=gobuilder /usr/local/go /usr/local/go
ENV GOPATH /go
ENV PATH /usr/local/go/bin:$GOPATH/bin:/usr/local/go/bin:$PATH

WORKDIR /src

ADD go.mod .
ADD go.sum .
ADD vendor/ vendor
ADD pkg/ pkg
ADD cmd/ cmd

RUN CGO_CPPFLAGS="-I/usr/local/include" \
    CGO_LDFLAGS="-L/usr/local/lib/x86_64-linux-gnu" \
    go build -v -a ./cmd/rc-steering