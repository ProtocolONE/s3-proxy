# S3 Proxy

S3 Proxy is a server proxy for S3 Amazon compatible storage servers. This proxy aim to protect your server credentials with limitation by size and mime types.

## Usage

```
docker run -it --rm --name \ 
    -e APP_S3_ACCESSKEYID=AKIAIOSFODNN7EXAMPLE \
    -e APP_S3_SECRETACCESSKEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
    -e APP_S3_REGION=eu-north-1 \
    -e APP_S3_BUCKET=uploads \
    -e APP_S3_ENDPOINT=http://localhost:9000 \
    -e APP_S3_ACL=public-read \
    p1hub/s3-proxy:latest proxy -c configs/local.yaml -b :8080
```  
This command up container with daemon on 8080 port and usage config files from container build (see `resources/configs/local.yaml`), 
for customize config just copy from this repository and mount file via `-v` option. 

*Open API* specification you can found here `spec/openapi.yaml`, use [Swagger UI](https://swagger.io/tools/swagger-ui/) to read and try it!

## Environment variables

| Name                                 | Required | Default                 | Description                               |
|:-------------------------------------|:--------:|:------------------------|:------------------------------------------|
| APP_WD                               | -        | application directory   | Work directory                            |
| APP_S3_ACCESSKEYID                   | true     | -                       | S3 access key for API access              |
| APP_S3_SECRETACCESSKEY               | true     | -                       | S3 secret key for API access              |
| APP_S3_TOKEN                         | -        | -                       | S3 token for API access                   |
| APP_S3_REGION                        | true     | -                       | The S3 region where the S3 bucket exists  |
| APP_S3_BUCKET                        | true     | -                       | The S3 bucket to be proxied with this app |
| APP_S3_ENDPOINT                      | true     | -                       | The endpoint for S3 API                   |
| APP_S3_ACL                           | true     | -                       | ACL mode for S3 request                   | 

## Development requirements
* GoLang 1.12+
* Docker

## How to start on Windows
* Install **MSYS2** https://www.msys2.org/ and append `bin` directory of MSYS2 to the `PATH` environment variable
* Install `docker`
* All commands should be execute in **DIND** container run `make dind`
* See **Development** section below for examine commands of make

## How to start on Mac, Linux
* Install `make`, `docker`
* There are two way for development: 
    * Native (recommend) 
        * Install `go` and see **Development** section below
    * DIND container (slow file system on Mac)
        * Run `make dind` and see **Development** section below

## Development
* For help, run `make`
* Init local env `make up` should run only once
* Download dependencies `make vendor`
* Generate source files from resource `make generate`
* Build and run application `make dev-build-up` (also it's usage for rebuild && recreate containers)

## Contributing
Please feel free to submit issues, fork the repository and send pull requests!

When submitting an issue, we ask that you please include a complete test function that demonstrates the issue. Extra credit for those using Testify to write the test code that demonstrates it.

## TODO
* Health check
* Tests (Travis)
* Build Docker image
* ACL (JWT)