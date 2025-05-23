FROM golang:1.15.3-alpine3.12 AS build
WORKDIR /
COPY . .
RUN CGO_ENABLED=0 go build -o /grabreflow -ldflags="-s -w"

# FROM selenium/standalone-chrome
FROM alpine:edge
WORKDIR /
# COPY ./config /config
COPY ./view /view
COPY --from=build /grabreflow /grabreflow
# COPY ./chromedriver /chromedriver
# wget https://chromedriver.storage.googleapis.com/97.0.4692.71/chromedriver_linux64.zip && unzip chromedriver_linux64.zip &&
RUN apk add --no-cache chromium chromium-chromedriver && apk add wqy-zenhei --update-cache --repository https://nl.alpinelinux.org/alpine/edge/testing
ENTRYPOINT [ "/grabreflow" ]