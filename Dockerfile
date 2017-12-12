FROM alpine:3.6
EXPOSE 8080

RUN apk add --no-cache ca-certificates && update-ca-certificates

COPY rootfs/k8s-gateway /usr/local/bin/k8s-gateway

CMD /usr/local/bin/k8s-gateway
