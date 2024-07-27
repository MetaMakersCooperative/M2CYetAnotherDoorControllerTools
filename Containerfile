# syntax = docker/dockerfile:1.4
ARG MOSQUITTO_VERSION=2.0.18
ARG MYSQL_VERSION=8.3.0
ARG GOLANG_VERSION=1.21.0
ARG ALPINE_VERSION=3.20.0
ARG MIGRATE_VERSION=v4.17.0

# =~=~=~=~=~=~= General Images =~=~=~=~=~=~=
FROM eclipse-mosquitto:${MOSQUITTO_VERSION} AS mosquitto
FROM mysql:${MYSQL_VERSION} AS mysql_database
FROM golang:${GOLANG_VERSION}-alpine AS golang
FROM alpine:${ALPINE_VERSION} AS alpine_linux
FROM migrate/migrate:${MIGRATE_VERSION} AS migrate

# =~=~=~=~=~=~= Password File Creation =~=~=~=~=~=~=
# NOTE: Only use this image in development environments!
FROM mosquitto AS password_gen

RUN touch /opt/passwd_file && chmod 0600 /opt/passwd_file
RUN mosquitto_passwd -b /opt/passwd_file door_one 'Door_One!1'
RUN mosquitto_passwd -b /opt/passwd_file door_two 'Door_Two!2'
RUN mosquitto_passwd -b /opt/passwd_file door_three 'Door_Three!3'
RUN mosquitto_passwd -b /opt/passwd_file door_four 'Door_Four!4'
RUN mosquitto_passwd -b /opt/passwd_file door_five 'Door_Five!5'
RUN mosquitto_passwd -b /opt/passwd_file porter 'BritishD00rMan!'
RUN mosquitto_passwd -b /opt/passwd_file access_list 'ACce55L12T!'

# =~=~=~=~=~=~= Mosquitto Run =~=~=~=~=~=~=
# NOTE: Only use this image in development environments!
FROM mosquitto AS mosquitto_broker

COPY --from=password_gen --chown=mosquitto:mosquitto /opt/passwd_file /mosquitto/passwd_file

# =~=~=~=~=~=~= Migration Container =~=~=~=~=~=~=
FROM migrate AS migrate_access_database

ENTRYPOINT [ "sh", "-c" ]
COPY ./migrations /migrations
CMD [ "exec migrate -verbose -database mysql://$DB_CONNECTION_URL -source file:///migrations up" ]

# =~=~=~=~=~=~= Go Build Container =~=~=~=~=~=~=
FROM golang AS build_porter

COPY ./porter /opt/porter
WORKDIR /opt/porter
RUN go build -o bin/porter main.go

# =~=~=~=~=~=~= Porter Base Container =~=~=~=~=~=~=
FROM alpine_linux AS porter_base
RUN addgroup -S porter && adduser -S porter -G porter
RUN mkdir -p /opt && chown porter:porter /opt
COPY --from=build_porter --chown=porter:porter /opt/porter/bin/porter /opt/porter
COPY ./LICENSE /opt/LICENSE

USER porter
WORKDIR /opt

# =~=~=~=~=~=~= Access List Container =~=~=~=~=~=~=
FROM porter_base AS porter_access_list

ENTRYPOINT [ "sh", "-c" ]
CMD [ "exec ./porter access_list" ]

# =~=~=~=~=~=~= Diary Container =~=~=~=~=~=~=
FROM porter_base AS porter_diary

ENTRYPOINT [ "sh", "-c" ]
CMD [ "exec ./porter diary" ]
