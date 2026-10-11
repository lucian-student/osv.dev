# Postgresql migration

Since currently i want to work on on the backend i need to figure how to start it and what code needs to change for me to migrate.
* Apparently it should be behind interface.


```
'buildx', 'build', '--load', '-t', 'gcr.io/oss-vdb/osv-server:latest', '-t', 'gcr.io/oss-vdb/osv-server:$COMMIT_SHA', '--target', 'api', '--build-context', 'bindings=../bindings', '-f', 'Dockerfile', '--cache-from', 'gcr.io/oss-vdb/osv-server:latest', '--pull', '.'
```
* --load -> Shorthand for "--output=type=docker"
* -t -> tag
* --target -> podle AI, to říká, kde se má zastavit z builděním.
* --build-context -> 
* -f -> 
* --cache-from -> 
* --pull


Takže jsem zjistil, že main backendu(api) je v ./go/cmd/api/main.go

Takže tam bych začal pátrat po tom možná jako naimplementovat backend pro postgres.