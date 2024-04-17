# Door Controller MQTT

## Authenication

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

## Development & Testing

### Migrations

```bash
podman compose up -d
# Enter the migrate container
podman compose exec migrate sh

# Within the container, run the following command
# NOTE: DB_CONNECTION_URL env should already be set
migrate -database "mysql://$DB_CONNECTION_URL" -source file:///migrations up
```

### Commands

Commands need to be ran within the `porter` directory.

```bash
# Publish an access list
go run main.go porter access_list -u "access_list" -p "ACce55L12T\!" -m mqtt://localhost:1883 -d "mellon:Y0USl-l@lL\!P@s5@tcp(localhost:3306)/access_system"

# Run Porter Diary
go run main.go porter diary -u "porter" -p "BritishD00rMan\!" -m mqtt://localhost:1883

# Run Porter Mimic
go run main.go porter mimic -u "door_one" -p "Door_One\!1" -m mqtt://localhost:1883
```

### Passwords

> REMEMBER! Do not bind a local directory to the container's `/mosquitto` directory.

```bash
# Add new user to file
podman run -it --rm -v ./mosquitto/passwd_file:/opt/passwd_file eclipse-mosquitto:2 mosquitto_passwd /opt/passwd_file <username>
```

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
