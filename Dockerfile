# syntax = docker/dockerfile:1.4
ARG MOSQUITTO_VERSION=2.0.18
ARG MYSQL_VERSION=8.3.0

# =~=~=~=~=~=~= General Images =~=~=~=~=~=~=
FROM eclipse-mosquitto:${MOSQUITTO_VERSION} as mosquitto
FROM mysql:${MYSQL_VERSION} as mysql

# =~=~=~=~=~=~= Password File Creation =~=~=~=~=~=~=
FROM mosquitto as password_gen

RUN touch /opt/passwd_file && chmod 0600 /opt/passwd_file
RUN mosquitto_passwd -b /opt/passwd_file door_one 'Door_One!1'
RUN mosquitto_passwd -b /opt/passwd_file door_two 'Door_Two!2'
RUN mosquitto_passwd -b /opt/passwd_file door_three 'Door_Three!3'
RUN mosquitto_passwd -b /opt/passwd_file door_four 'Door_Four!4'
RUN mosquitto_passwd -b /opt/passwd_file door_five 'Door_Five!5'
RUN mosquitto_passwd -b /opt/passwd_file porter 'BritishD00rMan!'

# =~=~=~=~=~=~= Mosquitto Run =~=~=~=~=~=~=
FROM mosquitto as mosquitto_broker

COPY --from=password_gen --chown=mosquitto:mosquitto /opt/passwd_file /mosquitto/passwd_file
