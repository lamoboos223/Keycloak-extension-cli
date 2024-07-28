## Pre-requisites

install the following:
- git
- java
- mvn



## USAGE

Set `KEYCLOAK_PATH` in environment variable.

```sh
export KEYCLOAK_PATH=/path/to/keycloak-server
```
### Build the Project

Keycloak Extension Manager
```sh
go build -o kem main.go
```

### Execute the installation of an extension

```sh
kem install --url=https://github.com/lamoboos223/keycloak-dummy-otp-extension
kem uninstall --file=keycloak-dummy-otp-extension.jar
kem list
kem --help
kem install --help
```



### Note

in this project we're assuming you're restarting your keycloak service like this

```sh
systemctl restart keycloak
```
