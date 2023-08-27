FROM golang:1.20.7 AS builder

# download modules
WORKDIR /atreus
COPY ./go.mod .
RUN go mod download

# build
FROM builder AS runner
ARG SERVICE_NAME

WORKDIR /atreus
COPY . .

RUN go build -o bin/${SERVICE_NAME} /atreus/app/${SERVICE_NAME}/service/cmd

# run
FROM debian:stable-slim
ARG SERVICE_NAME

ENV SERVICE_NAME ${SERVICE_NAME}

WORKDIR /app
COPY --from=runner /atreus/bin/${SERVICE_NAME} bin/
COPY --from=runner /atreus/configs/service/${SERVICE_NAME} configs/

CMD bin/${SERVICE_NAME} -conf configs