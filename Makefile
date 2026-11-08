.PHONY: all build test check flash clean

all:
	$(MAKE) clean
	$(MAKE) flash

build:
	@mkdir -p build
	go run ./cmd/build
	arm-none-eabi-size build/gyro-breakout-go.elf

test:
	go test ./...

check: test
	go vet ./...
	$(MAKE) build

flash: build
	openocd -f board/stm32f429discovery.cfg -c "program build/gyro-breakout-go.elf verify reset exit"

clean:
	rm -rf build
