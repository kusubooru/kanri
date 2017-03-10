# kanri - 管理
Management tools for Shimmie2.

## Features
* Safe approval
* Tag approval
* Image tag history
* Image tag history diff
* Tags diff

## Installation

```console
go get -u github.com/kusubooru/kanri
```

## Usage

Local example:

```console
kanri -dbconfig=username:password@(localhost:3306)/database?parseTime=true \
  -imagepath=/<path to images>/images \
  -thumbpath=/<path to image thumbs>/thumbs
```

Live example:

```console
kanri -http="localhost:8081" \
  -loginurl="/user_admin/login" \
  -dbconfig="username:password@(host:port)/database?parseTime=true" \
  -tlscert="/<TLS public key path>/cert.pem" \
  -tlskey="/<TLS private key path>/privkey.pem" \
  -imagepath=/<path to images>/images \
  -thumbpath=/<path to image thumbs>/thumbs
```

Development:

```console
go generate &&
go install &&
kanri -dbconfig="username:password@(localhost:3306)/database?parseTime=true" \
  -imagepath=/var/www/html/images \
  -thumbpath=/var/www/html/thumbs
```
