FROM golang:1.19 AS builder
WORKDIR /build
COPY ./src .
ARG GOOS
ARG GOARCH
RUN CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build -a -installsuffix cgo -o agent .

FROM ubuntu
RUN apt-get update
RUN apt-get install dnsutils -y
RUN apt-get install curl -y

# install psql
RUN apt-get install postgresql-client -y

RUN apt-get install unzip && \
    curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && \
    unzip awscliv2.zip && \
    ./aws/install

COPY --from=builder /build/agent /app/
WORKDIR /app
ENTRYPOINT ["./agent", "server"]