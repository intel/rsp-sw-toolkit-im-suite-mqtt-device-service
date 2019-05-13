FROM scratch

COPY /cmd /
COPY mqtt-device-service /

ENV APP_PORT=49982
EXPOSE $APP_PORT

ENTRYPOINT ["/mqtt-device-service"]
CMD ["--registry","--profile=docker","--confdir=/res"]
