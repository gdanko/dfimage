# dfimage
A small utility to extract a Dockerfile from a local Docker container image

## Introduction
Sometimes you want to see how a Docker image is built. This small utility will pretty accurately extract the entire contents of the Dockerfile from the image.

## How Does It Work?
Oh man, a lot goes on to do this and I will explain it later. I promise.

## Installation
Clone the repository and from within the repository directory, type `make build`. This will create a directory with the given value of `GOOS` and install the binary there. It will also create a tarball which will eventually be used for Homebrew formulae.

## Usage
```Usage:
  dfimage --image <image_name:tag> [--socket /path/to/docker.sock]
  dfimage extracts a Dockerfile from the specified image name and prints it to STDOUT.

Application Options:
  -d, --debug    Show debug information.
  -i, --image=   Specify the name of the image you want to inspect.
  -s, --socket=  Specify the path to the docker.sock file.
  -o, --outfile= Write the Dockerfile data to --outfile.
  -V, --version  Display version information and exit.

Help Options:
  -h, --help     Show this help message
  ```

The only required option is `-i` and this is the name of the image. If you don't specify a tag name, `latest` is assumed. The `-s` option should never be needed. It's only useful if the `docker.sock` file lives in a non-standard location.

## Example
```
$ dfimage -i rancher/klipper-helm:v0.8.3-build20240228
FROM <base image not found locally>
ADD file:d0764a717d1e9d0aff3fa84779b11bfa0afe4430dcb6b46d965b209167639ba0 in /
CMD ["/bin/sh"]
ARG BUILDDATE
LABEL buildDate=
RUN apk --no-cache upgrade
    && apk add -U --no-cache ca-certificates jq bash
    && adduser -D -u 1000 -s /bin/bash klipper-helm
WORKDIR /home/klipper-helm
COPY --chown=1000:1000dir:7927464555a2c7e5e5bec024c7b2092f99239aecb8aef5edf7457aafc7fc817a in /home/klipper-helm/.local/share/helm/plugins/
COPY multi:ed8926321cb4dc0a79f9ce07402c779c8917f6a6414a1e9f41bb08a551649bb2 in /usr/bin/
ENTRYPOINT ["entry"]
ENV STABLE_REPO_URL=https://charts.helm.sh/stable/
ENV TIMEOUT=
USER 1000
LABEL org.opencontainers.image.created=2024-02-29T00:00:41Z
LABEL org.opencontainers.image.revision=47d8af48899a1b48bbe9e2a749a3f6e4074201b2
LABEL org.opencontainers.image.source=https://github.com/k3s-io/klipper-helm.git
LABEL org.opencontainers.image.url=https://github.com/k3s-io/klipper-helm
```