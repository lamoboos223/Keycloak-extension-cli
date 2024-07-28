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

### Execute the installation of an extension

```sh
./Keycloak-extension-cli install --url=https://github.com/lamoboos223/keycloak-dummy-otp-extension
```



### Note

in this project we're assuming you're restarting your keycloak service like this

```sh
systemctl restart keycloak
```
