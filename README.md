# Door Controller MQTT

> This project is still a work in progress. Please use with caution until a stable release has been published!

This repository contains a CLI tool designed to be used with the [MetaMakersCooperative/M2C_Yet_Another_Door_Controller](https://github.com/MetaMakersCooperative/M2C_Yet_Another_Door_Controller) Arduino project. The tools help with testing, updating the card access list, and logging events published by the deployed door controllers.

## Building

```bash
cd porter
go build -o bin/porter main.go
```

## Commands

Commands need to be ran within the `porter` directory.

```bash
# Publish an access list
go run main.go access_list -u "access_list" -p "ACce55L12T\!" -m mqtt://localhost:1883 -d "mellon:Y0USl-l@lL\!P@s5@tcp(localhost:3306)/access_system"

# Run Porter Diary
go run main.go diary -u "porter" -p "BritishD00rMan\!" -m mqtt://localhost:1883

# Run Porter Mimic
go run main.go mimic -u "door_one" -p "Door_One\!1" -m mqtt://localhost:1883
```

## Development & Testing

### `compose.yml`

The provided `Containerfile` and `compose.yml` file are meant to be used for development. Do not use them in production environments.

To start the services, run the following command:

```bash
podman compose up -d
```

There are three services defined in the `compose.yml` file: `mosquitto`, `database`, and `migrate`.

> CAUTION! Do not directly bind a local directory to the `/mosquitto` directory. It likes to change the permissions and owners of files within that directory.

#### `mosquitto`

The `mosquitto` service runs [mosquitto](https://mosquitto.org/) as the MQTT broker. It's configured to listen on port `1883`.

The configuration used for development can be found within the `mosquitto` directory in the project's root.

##### Authentication

When the services are first built, a password file is created with the usernames and passwords listed in the table below. See the `Containerfile` for more information.

> WARNING! Do **not** use the provide password file for *any* environment save for testing & development!

| Username    | Password        |
| ----------- | --------------- |
| door_one    | Door_One!1      |
| door_two    | Door_Two!2      |
| door_three  | Door_Three!3    |
| door_four   | Door_Four!4     |
| door_five   | Door_Five!5     |
| access_list | ACce55L12T!     |
| porter      | BritishD00rMan! |


If more usernames/passwords are required, create a separate compose file and merge it with the one provided. To achieve this, create a file within the root of this project named, `compose.passwords.yml` containing the contents listed below:

```yml
---
services:
  mosquitto:
    volumes:
      - ./passwd_file:/mosquitto/passwd_file
```

To create the `passwd_file`, see mosquitto's documentation on [authentication methods](https://mosquitto.org/documentation/authentication-methods/) and the examples in the [Generate Password File](#generate-password-file) section.

#### `database`

> NOTE: The database schema is still a work in progress!

The `database` service runs the MySQL database that stores the list of active cards.

> In production, it's recommended that the credentials used by this program to be scoped to only allow the use of `SELECT` statements.

#### `migrate`

The `migrate` service's purpose is to make standing up a testing database a lot more convenient. See the [Migrations](#migrations) section for more details.

### Migrations

```bash
podman compose up -d
# Enter the migrate container
podman compose exec migrate sh

# Within the container, run the following command
# NOTE: DB_CONNECTION_URL env should already be set
migrate -database "mysql://$DB_CONNECTION_URL" -source file:///migrations up
```

## MQTT Broker Authenication

Contains command for creating a mosquitto `passwd_file` and managing that file's entries

> WARNING! Do **not** bind a local directory to the container's `/mosquitto` directory! It'll override and change file permissions and create a *huge* mess of the local directory.

### Generate Password File

```bash
# Create a new passwd file within the provided `<localdir>` and it'll contain an entry the the provided `<username>`
# NOTE: `-c` will override the file if it already exists! Use with caution.
podman run -it --rm -v ./<localdir>:/opt eclipse-mosquitto:2 mosquitto_passwd -c /opt/passwd_file <username>
```

### Add New User

```bash
# Add the provided user `<username>` to the passwd file located at `<localdir>`
podman run -it --rm -v ./<localdir>/passwd_file:/opt/passwd_file eclipse-mosquitto:2 mosquitto_passwd /opt/passwd_file <username>
```
