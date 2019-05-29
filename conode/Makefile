CONTAINER = stainless
IMAGE_NAME = c4dt/$(CONTAINER)
TAG = latest-$(shell date '+%Y%m%d')

# -s -w are for smaller binaries
# -X compiles the git tag into the binary
ldflags=-s -w -X main.gitTag=$(TAG)

all: docker

# Use this target to build from local source
docker: clean build/Dockerfile build/conode build/stainless.jar build/z3 build/cvc4
	docker build \
		--tag $(IMAGE_NAME):$(TAG) \
		--file build/Dockerfile \
		build
	docker tag $(IMAGE_NAME):$(TAG) $(IMAGE_NAME):latest

newdb:
	make -C ../src cothority-pull
	@echo "Compiling binaries"; \
	rm -f build/conodes/*db; \
	GO111MODULE=on go build -o build/conode; \
	cd ../src/cothority; \
		go build -o conode/bcadmin ./byzcoin/bcadmin; \
		go build -o conode/phapp ./personhood/phapp
	@echo "Starting nodes"; \
	cp ../src/cothority/conode/run_nodes.sh build; \
	( cd build; COTHORITY_ALLOW_INSECURE_ADMIN=true ./run_nodes.sh -v 2 -t -n 4 -s -d conodes > /dev/null & )
	@echo "Waiting for nodes to start up"; sleep 10; \
	echo "Creating new ByzCoin and printing out important variables"; echo; \
	rm -f *.cfg; \
	( ./bcadmin -c . create build/conodes/public.toml --interval 500ms; \
	  ./bcadmin latest --bc bc-*; \
	  ./bcadmin key -print key-* ) | egrep "^(ByzCoin|Admin|Private)" | tee variables.txt
	@echo; echo "Shutting down conodes"; pkill -x conode; rm build/conodes/running

docker_run:
	docker rm --force $(CONTAINER) || echo nothing to stop
	docker run \
		--rm \
		--interactive \
		--tty \
		--publish 9000-9007:9000-9007 \
		--name $(CONTAINER) $(IMAGE_NAME):latest

docker_stop:
	docker stop $(CONTAINER)

docker_start:
	docker start $(CONTAINER)

docker_clean:
	docker kill $(CONTAINER) || echo nothing to stop
	docker image ls $(IMAGE_NAME) --quiet | xargs docker image rm --force || echo done

# The suffix on conode exe is the result from: echo `uname -s`.`uname -m`
# so that we can find the right one in the wrapper script.
# This is in it's own rule because the Docker build needs it also.
build/conode:
	docker run \
		--rm \
		--volume "$$PWD/..":/usr/src/stainless \
		--volume $$(go env GOPATH)/pkg/mod:/go/pkg/mod \
		--workdir /usr/src/stainless/conode \
		--env CGO_ENABLED=1 \
		--env GO111MODULE=on \
		golang:1.12 go build -ldflags="$(ldflags)" -o $@

build/stainless.jar:
	wget --quiet --output-document $@ https://github.com/epfl-lara/smart/releases/download/v0.1s/stainless-0.1s-5a2b680.jar

build/z3:
	wget --quiet --output-document /tmp/z3.zip https://github.com/Z3Prover/z3/releases/download/z3-4.7.1/z3-4.7.1-x64-debian-8.10.zip && unzip -xp /tmp/z3.zip z3-4.7.1-x64-debian-8.10/bin/z3 > $@ && chmod +x $@ && rm /tmp/z3.zip

build/cvc4:
	wget --quiet --output-document $@ http://cvc4.cs.stanford.edu/downloads/builds/x86_64-linux-opt/cvc4-1.6-x86_64-linux-opt && chmod +x $@

clean:
	rm -rf build/conode
