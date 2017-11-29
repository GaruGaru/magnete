FROM golang:1.9

MAINTAINER Tommaso Garuglieri <garuglieritommaso@gmail.com>

RUN wget https://github.com/golang/dep/releases/download/v0.3.1/dep-linux-amd64 -O /usr/local/bin/dep && chmod +x /usr/local/bin/dep

WORKDIR /gopath/src/github.com/GaruGaru/magnete

ENV PORT 80
ENV GOPATH /gopath
ENV PATH $PATH:/usr/local/go/bin:$GOPATH/bin

COPY . ./

RUN dep ensure

RUN go build *.go

HEALTHCHECK --interval=5m --timeout=3s CMD curl -f http://127.0.0.1/probe

CMD ["/gopath/src/github.com/GaruGaru/magnete/main"]