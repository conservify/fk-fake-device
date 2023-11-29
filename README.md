# Overview

This is a Go application that pretends to be a FieldKit device on the local
network for testing the mobile application's communications. After starting it
will begin broadcasting over UDP so that the application can discover its
address and port number. It will then service requests on that port, returning
mock data.

## Prerequisites
- Go (https://go.dev/)


## Running the code

`go run .` in the termainal


#### TITLE:	
README for fk-fake-device
#### AUTHOR:	
Jacob Lewallen
#### EMAIL:	
jacob@conservify.org
