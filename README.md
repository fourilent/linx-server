linx-server 
======

Self-hosted file/media sharing website.  

### Demo
You can see what it looks like using the demo: [https://put.icu/](https://put.icu/)


### Clients
**Official**
- CLI: **linx-client** - [Source](https://github.com/andreimarcu/linx-client)

**Unofficial**
- Android: **LinxShare** - [Source](https://github.com/iksteen/LinxShare/) | [Google Play](https://play.google.com/store/apps/details?id=org.thegraveyard.linxshare)
- CLI: **golinx** - [Source](https://github.com/mutantmonkey/golinx)


### Features

- Display common filetypes (image, video, audio, markdown, pdf)  
- Display syntax-highlighted code with in-place editing
- Documented API with keys for restricting uploads
- File expiry, deletion key, file access key, and random filename options


### Screenshots
<img width="730" src="https://user-images.githubusercontent.com/4650950/76579039-03c82680-6488-11ea-8e23-4c927386fbd9.png" />

<img width="180" src="https://user-images.githubusercontent.com/4650950/76578903-771d6880-6487-11ea-8baf-a4a23fef4d26.png" /> <img width="180" src="https://user-images.githubusercontent.com/4650950/76578910-7be21c80-6487-11ea-9a0a-587d59bc5f80.png" /> <img width="180" src="https://user-images.githubusercontent.com/4650950/76578908-7b498600-6487-11ea-8994-ee7b6eb9cdb1.png" /> <img width="180" src="https://user-images.githubusercontent.com/4650950/76578907-7b498600-6487-11ea-8941-8f582bf87fb0.png" />


Getting started
-------------------

#### Using Docker
1. Create directories ```files``` and ```meta``` and run ```chown -R 65534:65534 meta && chown -R 65534:65534 files``` 
2. Create a config file (example provided in repo), we'll refer to it as __linx-server.conf__ in the following examples



Example running
```
docker run -p 8080:8080 -v /path/to/linx-server.conf:/data/linx-server.conf -v /path/to/meta:/data/meta -v /path/to/files:/data/files andreimarcu/linx-server -config /data/linx-server.conf
``` 

Example with docker-compose 
```
version: '2.2'
services:
  linx-server:
    container_name: linx-server
    image: andreimarcu/linx-server
    command: -config /data/linx-server.conf
    volumes:
      - /path/to/files:/data/files
      - /path/to/meta:/data/meta
      - /path/to/linx-server.conf:/data/linx-server.conf
    network_mode: bridge
    ports:
      - "8080:8080"
    restart: unless-stopped
```
Ideally, you would use a reverse proxy such as nginx or caddy to handle TLS certificates.

#### Using a binary release

1. Grab the latest binary from the [releases](https://github.com/andreimarcu/linx-server/releases), then run ```go install```
2. Run ```linx-server -config path/to/linx-server.conf```

  
Usage
-----

#### Configuration
All configuration options are accepted either as arguments or can be placed in a file as such (see example file linx-server.conf.example in repo):  
```ini
bind = 127.0.0.1:8080
sitename = myLinx
maxsize = 4294967296
maxexpiry = 86400
# ... etc
``` 
...and then run ```linx-server -config path/to/linx-server.conf```    

#### Options

|Option|Description
|------|-----------
| ```bind = 127.0.0.1:8080``` | what to bind to  (default is 127.0.0.1:8080)
| ```sitename = myLinx``` | the site name displayed on top (default is inferred from Host header)
| ```siteurl = https://mylinx.example.org/``` | the site url (default is inferred from execution context)
| ```selifpath = selif``` | path relative to site base url (the "selif" in mylinx.example.org/selif/image.jpg) where files are accessed directly (default: selif)
| ```maxsize = 4294967296``` | maximum upload file size in bytes (default 4GB)
| ```maxexpiry = 86400``` | maximum expiration time in seconds (default is 0, which is no expiry)
| ```allowhotlink = true``` | Allow file hotlinking
| ```nologs = true``` | (optionally) disable request logs in stdout
| ```force-random-filename = true``` | (optionally) force the use of random filenames
| ```custompagespath = custom_pages/``` | (optionally) specify path to directory containing markdown pages (must end in .md) that will be added to the site navigation (this can be useful for providing contact/support information and so on). For example, custom_pages/My_Page.md will become My Page in the site navigation 
| ```extra-footer-text = "..."``` | (optionally) Extra text above the footer for notices.
| ```max-duration-time = 0``` | Time till expiry for files over max-duration-size. (Default is 0 for no-expiry.)
| ```max-duration-size = 4294967296``` | Size of file before max-duration-time is used to determine expiry max time. (Default is 4GB)
| ```disable-access-key = true``` | Disables access key usage. (Default is false.)
| ```default-random-filename = true``` | Makes it so the random filename is not default if set false. (Default is true.)


#### Cleaning up expired files
When files expire, access is disabled immediately, but the files and metadata
will persist on disk until someone attempts to access them. You can set the following option to run cleanup every few minutes. This can also be done using a separate utility found the linx-cleanup directory.


|Option|Description
|------|-----------
| ```cleanup-every-minutes = 5``` | How often to clean up expired files in minutes (default is 0, which means files will be cleaned up as they are accessed)


#### Require API Keys for uploads

|Option|Description
|------|-----------
| ```authfile = path/to/authfile``` | (optionally) require authorization for upload/delete by providing a newline-separated file of scrypted auth keys
| ```basicauth = true``` | (optionally) allow basic authorization to upload or paste files from browser when `-authfile` is enabled. When uploading, you will be prompted to enter a user and password - leave the user blank and use your auth key as the password

A helper utility ```linx-genkey``` is provided which hashes keys to the format required in the auth files.

#### Storage backends
The following storage backends are available:

|Name|Notes|Options
|----|-----|-------
|LocalFS|Enabled by default, this backend uses the filesystem|```filespath = files/``` -- Path to store uploads (default is files/)<br />```metapath = meta/``` -- Path to store information about uploads (default is meta/)|
|S3|Use with any S3-compatible provider.<br> This implementation will stream files through the linx instance (every download will request and stream the file from the S3 bucket). File metadata will be stored as tags on the object in the bucket.<br><br>For high-traffic environments, one might consider using an external caching layer such as described [in this article](https://blog.sentry.io/2017/03/01/dodging-s3-downtime-with-nginx-and-haproxy.html).|```s3-endpoint = https://...``` -- S3 endpoint<br>```s3-region = us-east-1``` -- S3 region<br>```s3-bucket = mybucket``` -- S3 bucket to use for files and metadata<br>```s3-force-path-style = true``` (optional) -- force path-style addresing (e.g. https://<span></span>s3.amazonaws.com/linx/example.txt)<br><br>Environment variables to provide:<br>```AWS_ACCESS_KEY_ID``` -- the S3 access key<br>```AWS_SECRET_ACCESS_KEY ``` -- the S3 secret key<br>```AWS_SESSION_TOKEN``` (optional) -- the S3 session token|


#### SSL with built-in server 
|Option|Description
|------|-----------
| ```certfile = path/to/your.crt``` | Path to the ssl certificate (required if you want to use the https server)
| ```keyfile = path/to/your.key``` | Path to the ssl key (required if you want to use the https server)

#### Use with http proxy 
|Option|Description
|------|-----------
| ```realip = true``` | let linx-server know you (nginx, etc) are providing the X-Real-IP and/or X-Forwarded-For headers.

Deployment
----------
Linx-server supports being deployed in a subdirectory (ie. example.com/mylinx/) as well as on its own (example.com/).

#### 1. Using the built-in https server
Run linx-server with the ```certfile = path/to/cert.file``` and ```keyfile = path/to/key.file``` options.

#### 2. Using the built-in http server
Run linx-server normally.

Development
-----------
Any help is welcome, PRs will be reviewed and merged accordingly.  
The official IRC channel is #linx on irc.oftc.net  

1. ```go get -u github.com/andreimarcu/linx-server ```
2. ```cd $GOPATH/src/github.com/andreimarcu/linx-server ```
3. ```go build && ./linx-server```


License
-------
Copyright (C) 2015 Andrei Marcu

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.

Author
-------
Andrei Marcu, https://andreim.net/
