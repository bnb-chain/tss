build_trust:
	if [ ! -d "./wallet-core" ]; then \
		git clone https://github.com/ackratos/wallet-core.git; \
	fi
	cd wallet-core && git checkout tss && \
	./bootstrap.sh && make -C build

build: build_trust
	go build -tags=deluxe