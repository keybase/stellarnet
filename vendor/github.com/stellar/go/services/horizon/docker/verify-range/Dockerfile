FROM ubuntu:20.04

MAINTAINER Bartek Nowotarski <bartek@stellar.org>

ENV STELLAR_CORE_VERSION 18.0.3-746.f3baea6.focal
# to remove tzdata interactive flow
ENV DEBIAN_FRONTEND=noninteractive

ADD dependencies /
RUN ["chmod", "+x", "dependencies"]
RUN /dependencies

ADD stellar-core.cfg /

ADD start /
RUN ["chmod", "+x", "start"]

ENTRYPOINT ["/start"]
