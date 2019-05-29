FROM debian:stretch-slim
WORKDIR /root/
COPY conodes /root/conodes

# See https://bugs.debian.org/cgi-bin/bugreport.cgi?bug=863199
RUN mkdir -p /usr/share/man/man1
# libgomp is needed for z3
RUN apt-get update; \
    apt-get install --no-install-recommends --yes \
    procps \
    openjdk-8-jre-headless \
    libgomp1 \
    ; apt-get clean
# Backports needed for 'npm'...
RUN echo 'deb http://deb.debian.org/debian stretch-backports main' >> /etc/apt/sources.list
RUN apt-get update; \
    apt-get install --target-release stretch-backports --no-install-recommends --yes \
    npm \
    ; apt-get clean
RUN npm install --global solc@0.5.8

COPY run_nodes.sh .
COPY conode /root/conode
COPY stainless.jar /root/
COPY stainless z3 cvc4 /usr/local/bin/

EXPOSE 9000 9001 9002 9003 9004 9005 9006 9007

CMD COTHORITY_ALLOW_INSECURE_ADMIN=true ./run_nodes.sh -n 4 -v 2 -c -d /root/conodes
