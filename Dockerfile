# Dockerfile similar to the azure provider https://github.com/Azure/secrets-store-csi-driver-provider-azure/blob/bb062669205cc5419bf3624b7f008b126fdd0ca1/Dockerfile
FROM golang:1.13.4-alpine as builder
RUN apk add --update make
ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go
COPY . /go/src/github.com/freenowtech/secrets-store-csi-driver-provider-spring-cloud-config
WORKDIR /go/src/github.com/freenowtech/secrets-store-csi-driver-provider-spring-cloud-config
RUN make build

FROM alpine:3.10.3
RUN apk add --no-cache bash
COPY --from=builder /go/src/github.com/freenowtech/secrets-store-csi-driver-provider-spring-cloud-config/secrets-store-csi-driver-provider-spring-cloud-config /bin/
COPY --from=builder /go/src/github.com/freenowtech/secrets-store-csi-driver-provider-spring-cloud-config/install.sh /bin/install_spring-cloud-config_provider.sh
RUN chmod a+x /bin/secrets-store-csi-driver-provider-spring-cloud-config

ENTRYPOINT ["/bin/install_spring-cloud-config_provider.sh"]

