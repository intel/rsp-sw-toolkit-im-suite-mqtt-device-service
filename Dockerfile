FROM scratch

COPY /cmd /
COPY gateway-device-service /

ENV APP_PORT=49982
EXPOSE $APP_PORT

ENTRYPOINT ["/gateway-device-service"]
CMD ["--registry","--profile=docker","--confdir=/res"]
