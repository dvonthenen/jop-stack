FROM ubuntu:14.04
MAINTAINER David vonThenen <davidvonthenen@gmail.com>

RUN apt-get update -q
RUN apt-get -y install curl dnsutils

ADD simple /simple
ENTRYPOINT /simple
