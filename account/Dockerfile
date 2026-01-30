#go test
#author ebillow

#step 1
FROM golang:1.19.7-alpine3.17 as builder

WORKDIR /go/cache
ADD go.mod .
ADD go.sum .
RUN go mod download

WORKDIR /app
ADD . .

#RUN go build -ldflags "-s -w" -tags=release    #去掉调试信息，符号表
RUN cd account/ && go build -ldflags "-w" -tags=release

#step 2
FROM alpine:3.17 as runer
#ENV TZ=Asia/Shanghai
#RUN apk add --no-cache tzdata && ln -sf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

ARG SVN_VERSION
LABEL svn-version=$SVN_VERSION

FROM runer
COPY --from=builder /app/account/account /account
COPY --from=builder /app/account/IP2LOCATION-LITE-DB1.BIN /IP2LOCATION-LITE-DB1.BIN

CMD ["/account"]
